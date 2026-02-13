package wav

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/encoding/g711"
	"github.com/SladkyCitron/resona/encoding/pcm"
)

// Encoder represents the encoder for the WAVE file format.
//
// The caller retains ownership of the writer; it will not be closed automatically.
type Encoder struct {
	w         io.WriteSeeker
	format    afmt.Format
	sampleFmt afmt.SampleFormat
	wavFormat uint16

	enc         aio.SampleWriter
	dataWritten uint32
}

// NewEncoder creates a new [Encoder] for the WAVE file format (not extensible).
//
// The caller retains ownership of the writer; it will not be closed automatically.
//
// # Audio Formats
//
// wavFormat must be one of [FormatInt], [FormatFloat], [FormatAlaw], or [FormatUlaw].
// Unsupported formats will return an error.
//
// When using A-law or U-law, the sample format is ignored and the bit depth is set to 8.
func NewEncoder(w io.WriteSeeker, format afmt.Format, sampleFmt afmt.SampleFormat, wavFormat uint16) (*Encoder, error) {
	e := &Encoder{
		w:         w,
		format:    format,
		sampleFmt: sampleFmt,
		wavFormat: wavFormat,
	}

	switch wavFormat {
	case FormatInt, FormatFloat:
		e.enc = pcm.NewEncoder(w, sampleFmt)
	case FormatAlaw:
		sampleFmt.BitDepth = 8 // A-law is always 8-bit
		e.enc = g711.NewAlawEncoder(w)
	case FormatUlaw:
		sampleFmt.BitDepth = 8 // U-law is always 8-bit
		e.enc = g711.NewUlawEncoder(w)
	default:
		return nil, fmt.Errorf("unknown or unsupported audio format: %v", wavFormat)
	}

	if err := e.writeHeader(); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	return e, nil
}

func (e *Encoder) writeHeader() error {
	// Write "RIFF"
	if _, err := e.w.Write([]byte("RIFF")); err != nil {
		return err
	}

	// Write file size (placeholder)
	if err := binary.Write(e.w, binary.LittleEndian, uint32(0xFFFFFFFF)); err != nil {
		return err
	}

	// Write "WAVEfmt "
	if _, err := e.w.Write([]byte("WAVEfmt ")); err != nil {
		return err
	}

	// Write fmt  chunk size
	if err := binary.Write(e.w, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}

	// Write format
	if err := binary.Write(e.w, binary.LittleEndian, e.wavFormat); err != nil {
		return err
	}

	// Write channels
	if err := binary.Write(e.w, binary.LittleEndian, int16(e.format.NumChannels)); err != nil {
		return err
	}

	// Write sample rate
	if err := binary.Write(e.w, binary.LittleEndian, int32(e.format.SampleRate.Hertz())); err != nil {
		return err
	}

	// Write bytes per second
	byterate := int32(int(e.format.SampleRate.Hertz()) * e.format.NumChannels * e.sampleFmt.BytesPerSample())
	if err := binary.Write(e.w, binary.LittleEndian, byterate); err != nil {
		return err
	}

	// Write bytes per frame
	if err := binary.Write(e.w, binary.LittleEndian, int16(e.sampleFmt.BytesPerFrame(e.format.NumChannels))); err != nil {
		return err
	}

	// Write bit depth
	if err := binary.Write(e.w, binary.LittleEndian, int16(e.sampleFmt.BitDepth)); err != nil {
		return err
	}

	// Write "data"
	if _, err := e.w.Write([]byte("data")); err != nil {
		return err
	}

	// Write data chunk size (placeholder)
	if err := binary.Write(e.w, binary.LittleEndian, uint32(0xFFFFFFFF)); err != nil {
		return err
	}

	return nil
}

// WriteSamples encodes and writes samples.
func (e *Encoder) WriteSamples(p []float32) (int, error) {
	n, err := e.enc.WriteSamples(p)
	e.dataWritten += uint32(n)
	return n, err
}

// Close finalizes the encoding process and writes the length value.
//
// It will NOT close the underlying writer, even if it implements [io.Closer].
// Closing the underlying writer is the owner's responsibility.
func (e *Encoder) Close() error {
	dataSize := e.dataWritten * uint32(e.sampleFmt.BytesPerSample())
	riffSize := 36 + dataSize // 4 + (8 + fmt chunk) + (8 + data chunk)

	// Patch RIFF size
	if _, err := e.w.Seek(4, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to RIFF size: %w", err)
	}
	if err := binary.Write(e.w, binary.LittleEndian, riffSize); err != nil {
		return fmt.Errorf("failed to write RIFF size: %w", err)
	}

	// Patch data chunk size
	if _, err := e.w.Seek(40, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to data chunk size: %w", err)
	}
	if err := binary.Write(e.w, binary.LittleEndian, dataSize); err != nil {
		return fmt.Errorf("failed to write data chunk size: %w", err)
	}

	if _, err := e.w.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to restore file position: %w", err)
	}

	return nil
}
