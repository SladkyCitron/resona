package qoa

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
)

// Encoder represents the encoder for the QOA file format.
//
// It accumulates samples written via [Encoder.WriteSamples] until it has
// 256 samples per channel, at which point it encodes and writes a frame to the underlying writer.
//
// The caller retains ownership of the writer; it will not be closed automatically.
type Encoder struct {
	w        io.WriteSeeker
	format   afmt.Format
	samples  uint32
	lmsState []lms
	buf      []float32
}

// NewEncoder creates a new [Encoder] for QOA format.
func NewEncoder(w io.WriteSeeker, format afmt.Format) (*Encoder, error) {
	e := &Encoder{
		w:      w,
		format: format,
	}

	if format.SampleRate == 0 || uint32(format.SampleRate.Hertz()) > 0xffffff || format.NumChannels == 0 || format.NumChannels > maxChannels {
		return nil, fmt.Errorf("qoa: invalid format: %v", format)
	}

	// set initial LMS state to {0, 0, -1, 2} for each channel
	e.lmsState = make([]lms, format.NumChannels)
	for ch := range e.lmsState {
		e.lmsState[ch].weights = [lmsLen]int16{0, 0, -(1 << 13), (1 << 14)}
		e.lmsState[ch].history = [lmsLen]int16{0, 0, 0, 0}
	}

	if err := e.writeHeader(); err != nil {
		return nil, fmt.Errorf("qoa: failed to write header: %w", err)
	}

	return e, nil
}

func (e *Encoder) writeHeader() error {
	// write magic
	if _, err := e.w.Write([]byte(magic)); err != nil {
		return err
	}

	// write samples (placeholder)
	if err := binary.Write(e.w, binary.BigEndian, uint32(0)); err != nil {
		return err
	}

	return nil
}

func (e *Encoder) encodeFrame(frame []float32) error {
	frameLen := len(frame) / e.format.NumChannels
	slices := (frameLen + sliceLen - 1) / sliceLen
	size := frameSize(uint32(e.format.NumChannels), uint32(slices))
	var prevScalefactor [maxChannels]int

	sampleRate := uint32(e.format.SampleRate.Hertz()) & 0xffffff
	hdr := uint64(e.format.NumChannels)<<56 | uint64(sampleRate)<<32 | uint64(frameLen)<<16 | uint64(size)
	if err := binary.Write(e.w, binary.BigEndian, hdr); err != nil {
		return fmt.Errorf("qoa: failed to write frame header: %w", err)
	}

	for ch := range e.format.NumChannels {
		// write LMS state
		var history uint64
		var weights uint64
		for i := range lmsLen {
			history = (history << 16) | (uint64(e.lmsState[ch].history[i]) & 0xffff)
			weights = (weights << 16) | (uint64(e.lmsState[ch].weights[i]) & 0xffff)
		}
		if err := binary.Write(e.w, binary.BigEndian, history); err != nil {
			return fmt.Errorf("qoa: failed to write LMS history: %w", err)
		}
		if err := binary.Write(e.w, binary.BigEndian, weights); err != nil {
			return fmt.Errorf("qoa: failed to write LMS weights: %w", err)
		}
	}

	for sampleIndex := 0; sampleIndex < frameLen; sampleIndex += sliceLen {
		for ch := range e.format.NumChannels {
			_sliceLen := clamp(sliceLen, 0, frameLen-sampleIndex)
			sliceStart := sampleIndex*e.format.NumChannels + ch
			sliceEnd := (sampleIndex+_sliceLen)*e.format.NumChannels + ch

			var bestRank uint64 = 1<<64 - 1
			var bestSlice uint64
			var bestLMS lms
			var bestScalefactor int

			for sfi := range 16 {
				scalefactor := (sfi + prevScalefactor[ch]) & (16 - 1)

				_lms := e.lmsState[ch]
				slice := uint64(scalefactor)
				var currentRank uint64

				for si := sliceStart; si < sliceEnd; si += e.format.NumChannels {
					sample := int(frame[si] * (1<<15 - 1))
					predicted := _lms.predict()

					residual := sample - predicted
					scaled := div(residual, scalefactor)
					clamped := clamp(scaled, -8, 8)
					quantized := int(quantTab[clamped+8])
					dequantized := int(dequantTab[scalefactor][quantized])
					reconstructed := clampS16(predicted + dequantized)

					weightsPenalty := max((int(_lms.weights[0]*_lms.weights[0]+
						_lms.weights[1]*_lms.weights[1]+
						_lms.weights[2]*_lms.weights[2]+
						_lms.weights[3]*_lms.weights[3])>>18)-0x8ff, 0)

					var err int64 = int64(sample) - int64(reconstructed)
					errSq := err * err

					currentRank += uint64(int(errSq) + weightsPenalty*weightsPenalty)
					if currentRank > bestRank {
						break
					}

					_lms.update(reconstructed, int16(dequantized))
					slice = (slice << 3) | uint64(quantized)
				}

				if currentRank < bestRank {
					bestRank = currentRank
					bestSlice = slice
					bestLMS = _lms
					bestScalefactor = scalefactor
				}
			}

			prevScalefactor[ch] = bestScalefactor

			e.lmsState[ch] = bestLMS

			bestSlice <<= (sliceLen - _sliceLen) * 3
			if err := binary.Write(e.w, binary.BigEndian, bestSlice); err != nil {
				return fmt.Errorf("qoa: failed to write slice: %w", err)
			}
		}
	}

	return nil
}

// WriteSamples accumulates samples until it has 256 samples per channel,
// at which point it encodes and writes a frame to the underlying writer.
func (e *Encoder) WriteSamples(p []float32) (int, error) {
	written := len(p)
	e.buf = append(e.buf, p...)

	frameSize := 256 * e.format.NumChannels
	for len(e.buf) >= frameSize {
		frame := e.buf[:frameSize]
		if err := e.encodeFrame(frame); err != nil {
			return written, fmt.Errorf("qoa: failed to encode frame: %w", err)
		}
		e.buf = e.buf[frameSize:]
		e.samples += 256
	}

	return written, nil
}

// Close encodes any remaining frames and writes the data size.
//
// It will NOT close the underlying writer, even if it implements [io.Closer].
// Closing the underlying writer is the owner's responsibility.
func (e *Encoder) Close() error {
	if len(e.buf) != 0 {
		frameSize := 256 * e.format.NumChannels
		frame := e.buf[:frameSize]
		if err := e.encodeFrame(frame); err != nil {
			return fmt.Errorf("qoa: failed to encode frame: %w", err)
		}
	}

	// patch samples
	if _, err := e.w.Seek(int64(len(magic)), io.SeekStart); err != nil {
		return fmt.Errorf("qoa: failed to seek to samples: %w", err)
	}
	if err := binary.Write(e.w, binary.BigEndian, e.samples); err != nil {
		return fmt.Errorf("qoa: failed to write sampled: %w", err)
	}

	// seek to start
	if _, err := e.w.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("qoa: failed to seek to start: %w", err)
	}

	return nil
}
