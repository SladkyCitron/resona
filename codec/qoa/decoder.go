package qoa

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/codec"
	"github.com/SladkyCitron/resona/freq"
)

// Decoder represents the decoder for the QOA file format.
// It implements codec.Decoder.
type Decoder struct {
	r      io.Reader
	seeker io.Seeker

	samples    uint32 // samples per channel
	channels   uint8
	sampleRate uint32

	lmsState []lms

	frameBuf    []int16
	framePos    int
	readSamples int64 // read frames, not samples, but in the context of QOA, "frame" means something different

	frameOffsets []frameIndex
}

type frameIndex struct {
	offset  int64
	samples uint16
}

func NewDecoder(r io.Reader) (codec.Decoder, error) {
	d := &Decoder{r: r}
	d.seeker = r.(io.Seeker)

	// Read magic
	var buf [len(magic)]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, err
	}
	if string(buf[:]) != magic {
		return nil, errors.New("qoa: invalid header")
	}

	// Read samples
	if err := binary.Read(r, binary.BigEndian, &d.samples); err != nil {
		return nil, fmt.Errorf("qoa: failed to read samples per channel: %w", err)
	}

	if err := d.ensureFrameOffsets(); err != nil {
		return nil, err
	}

	if err := d.init(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Decoder) ensureFrameOffsets() error {
	if d.seeker == nil {
		return nil
	}

	d.frameOffsets = make([]frameIndex, int(math.Ceil(float64(d.samples)/(256*20))))

	// populate frame offsets
	for i := range d.frameOffsets {
		byteOffset, err := d.seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("qoa: failed to get frame byte offset: %w", err)
		}

		var hdr [8]byte
		if _, err := io.ReadFull(d.r, hdr[:]); err != nil {
			return fmt.Errorf("qoa: failed to read frame header: %w", err)
		}
		samples := uint16(hdr[4])<<8 | uint16(hdr[5])

		d.frameOffsets[i] = frameIndex{offset: byteOffset, samples: samples}

		// skip frame data
		fsize := uint16(hdr[6])<<8 | uint16(hdr[7])
		if _, err := io.CopyN(io.Discard, d.r, int64(fsize-8)); err != nil {
			return fmt.Errorf("qoa: failed to skip frame data: %w", err)
		}
	}

	// seek back to start
	// magic (4 bytes) + samples (4 bytes) = 8 bytes
	_, err := d.seeker.Seek(int64(len(magic))+4, io.SeekStart)
	return err
}

func (d *Decoder) init() error {
	// Read first frame, get channels and sample rate, and fill buffer
	var hdr [8]byte
	if _, err := io.ReadFull(d.r, hdr[:]); err != nil {
		return fmt.Errorf("qoa: failed to read frame header: %w", err)
	}

	d.channels = hdr[0]
	d.sampleRate = uint32(hdr[1])<<16 | uint32(hdr[2])<<8 | uint32(hdr[3]) // uint24 big endian
	samples := uint16(hdr[4])<<8 | uint16(hdr[5])
	fsize := uint16(hdr[6])<<8 | uint16(hdr[7])

	if d.channels == 0 || d.samples == 0 || d.sampleRate == 0 {
		return errors.New("qoa: invalid frame header values")
	}

	d.lmsState = make([]lms, d.channels)
	d.frameBuf = make([]int16, int(samples)*int(d.channels))

	frameBytes := make([]byte, fsize-8)
	if _, err := io.ReadFull(d.r, frameBytes); err != nil {
		return fmt.Errorf("qoa: failed to read frame: %w", err)
	}

	offset := 0

	// read LMS state
	for ch := range d.channels {
		// history
		history := binary.BigEndian.Uint64(frameBytes[offset:])
		weights := binary.BigEndian.Uint64(frameBytes[offset+8:])
		for i := range lmsLen {
			d.lmsState[ch].history[i] = int16(history >> 48)
			history <<= 16
			d.lmsState[ch].weights[i] = int16(weights >> 48)
			weights <<= 16
		}
		offset += 16
	}

	// decode all slices
	for sampleIdx := uint32(0); sampleIdx < uint32(samples); sampleIdx += sliceLen {
		for ch := range d.channels {
			slice := binary.BigEndian.Uint64(frameBytes[offset:])
			offset += 8

			var scalefactor int = int((slice >> 60) & 0xf)
			slice <<= 4

			sliceStart := int(sampleIdx)*int(d.channels) + int(ch)
			sliceEnd := clamp(int(sampleIdx)+sliceLen, 0, int(samples))*int(d.channels) + int(ch)

			for si := sliceStart; si < sliceEnd; si += int(d.channels) {
				predicted := d.lmsState[ch].predict()
				quantized := int((slice >> 61) & 0x7)
				dequantized := dequantTab[scalefactor][quantized]
				reconstructed := clampS16(predicted + int(dequantized))

				d.frameBuf[si] = reconstructed
				slice <<= 3

				d.lmsState[ch].update(reconstructed, dequantized)
			}
		}
	}

	d.readSamples += int64(samples)
	d.framePos = 0

	return nil
}

// decoding (again)

func (d *Decoder) readFrame() error {
	var hdr [8]byte
	if _, err := io.ReadFull(d.r, hdr[:]); err != nil {
		return fmt.Errorf("qoa: failed to read frame header: %w", err)
	}

	channels := hdr[0]
	sampleRate := uint32(hdr[1])<<16 | uint32(hdr[2])<<8 | uint32(hdr[3]) // uint24 big endian
	samples := uint16(hdr[4])<<8 | uint16(hdr[5])
	fsize := uint16(hdr[6])<<8 | uint16(hdr[7])

	dataSize := uint32(fsize) - 8 - lmsLen*4*uint32(channels)
	numSlices := dataSize / 8
	maxTotalSamples := numSlices * sliceLen

	if channels != d.channels || sampleRate != d.sampleRate || uint32(samples) > maxTotalSamples {
		return errors.New("qoa: invalid frame header values")
	}

	frameBytes := make([]byte, fsize-8)
	if _, err := io.ReadFull(d.r, frameBytes); err != nil {
		return fmt.Errorf("qoa: failed to read frame: %w", err)
	}

	offset := 0

	// read LMS state
	for ch := range d.channels {
		// history
		history := binary.BigEndian.Uint64(frameBytes[offset:])
		weights := binary.BigEndian.Uint64(frameBytes[offset+8:])
		for i := range lmsLen {
			d.lmsState[ch].history[i] = int16(history >> 48)
			history <<= 16
			d.lmsState[ch].weights[i] = int16(weights >> 48)
			weights <<= 16
		}
		offset += 16
	}

	// decode all slices
	for sampleIdx := uint32(0); sampleIdx < uint32(samples); sampleIdx += sliceLen {
		for ch := range d.channels {
			slice := binary.BigEndian.Uint64(frameBytes[offset:])
			offset += 8

			var scalefactor int = int((slice >> 60) & 0xf)
			slice <<= 4

			sliceStart := int(sampleIdx)*int(d.channels) + int(ch)
			sliceEnd := clamp(int(sampleIdx)+sliceLen, 0, int(samples))*int(d.channels) + int(ch)

			for si := sliceStart; si < sliceEnd; si += int(d.channels) {
				predicted := d.lmsState[ch].predict()
				quantized := int((slice >> 61) & 0x7)
				dequantized := dequantTab[scalefactor][quantized]
				reconstructed := clampS16(predicted + int(dequantized))

				d.frameBuf[si] = reconstructed
				slice <<= 3

				d.lmsState[ch].update(reconstructed, dequantized)
			}
		}
	}

	d.readSamples += int64(samples)
	d.framePos = 0

	return nil
}

// Format returns the audio stream format.
func (d *Decoder) Format() afmt.Format {
	return afmt.Format{
		SampleRate:  freq.Frequency(d.sampleRate) * freq.Hertz,
		NumChannels: int(d.channels),
	}
}

// SampleFormat returns the sample format.
func (d *Decoder) SampleFormat() afmt.SampleFormat {
	return afmt.SampleFormat{
		BitDepth: 16,
		Encoding: afmt.SampleEncodingInt,
		Endian:   binary.BigEndian,
	}
}

// Len returns the total number of frames.
func (d *Decoder) Len() int {
	return int(d.samples)
}

// ReadSamples reads float32 samples into p.
// It returns the number of samples read and/or an error.
func (d *Decoder) ReadSamples(p []float32) (int, error) {
	n := 0
	for n < len(p) {
		if d.framePos >= len(d.frameBuf) {
			// load next frame
			if err := d.readFrame(); err != nil {
				if errors.Is(err, io.EOF) {
					return n, io.EOF
				}
				return n, err
			}
		}

		// copy as many samples as possible, and convert int16 => float32
		remaining := len(d.frameBuf) - d.framePos
		toCopy := min(remaining, len(p)-n)
		for i := range toCopy {
			p[n+i] = float32(d.frameBuf[d.framePos+i]) / (1<<15 - 1)
		}
		d.framePos += toCopy
		n += toCopy
	}
	return n, nil
}

// Seek seeks to the specified frame.
// It returns the new offset relative to the start and/or an error.
// It will return an error if the source is not an [io.Seeker].
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	// special case
	if whence == io.SeekCurrent && offset == 0 {
		return d.readSamples, nil
	}

	if d.seeker == nil {
		return 0, errors.New("qoa: resource is not seekable")
	}

	var targetSample int64
	switch whence {
	case io.SeekStart:
		targetSample = offset
	case io.SeekCurrent:
		targetSample = int64(d.framePos) + offset
	case io.SeekEnd:
		targetSample = int64(d.samples) + offset
	default:
		return 0, errors.New("qoa: invalid whence")
	}

	if targetSample < 0 || targetSample > int64(d.samples) {
		return 0, errors.New("qoa: seek out of bounds")
	}

	d.readSamples = targetSample

	// find the frame
	var frameIdx int
	for i := range d.frameOffsets {
		if int64(d.frameOffsets[i].samples) > targetSample {
			frameIdx = i
			break
		}
		targetSample -= int64(d.frameOffsets[i].samples)
	}

	// seek to frame
	_, err := d.seeker.Seek(d.frameOffsets[frameIdx].offset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("qoa: failed to seek: %w", err)
	}

	if err := d.readFrame(); err != nil {
		return 0, err
	}

	d.framePos = int(targetSample) * int(d.channels)

	return int64(d.framePos), nil
}

func init() {
	codec.RegisterFormat("qoa", magic, NewDecoder)
}
