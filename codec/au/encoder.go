package au

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/encoding/g711"
	"github.com/SladkyCitron/resona/encoding/pcm"
)

// Encoder represents the encoder for the AU file format.
//
// The caller retains ownership of the writer; it will not be closed automatically.
type Encoder struct {
	w           io.WriteSeeker
	format      afmt.Format
	encoding    uint32
	extraData   []byte
	enc         aio.SampleWriter
	dataWritten int
}

// NewEncoder creates a new [Encoder] for AU format.
func NewEncoder(w io.WriteSeeker, format afmt.Format, encoding uint32, extraData []byte) (*Encoder, error) {
	e := &Encoder{
		w:         w,
		format:    format,
		encoding:  encoding,
		extraData: extraData,
	}

	if err := e.writeHeader(); err != nil {
		return nil, fmt.Errorf("au: failed to write header: %w", err)
	}

	if encoding < 1 || (encoding > 7 && encoding != 27) {
		return nil, fmt.Errorf("au: unsupported encoding %d", encoding)
	}

	switch encoding {
	case Ulaw:
		e.enc = g711.NewUlawEncoder(w)
	case Alaw:
		e.enc = g711.NewAlawEncoder(w)
	default:
		e.enc = pcm.NewEncoder(w, e.sampleFormat())
	}

	return e, nil

}

func (e *Encoder) sampleFormat() afmt.SampleFormat {
	format := afmt.SampleFormat{Endian: binary.BigEndian}
	switch e.encoding {
	case Ulaw:
		format.BitDepth = 8
		format.Encoding = afmt.SampleEncodingUnknown
	case LPCMInt8:
		format.BitDepth = 8
		format.Encoding = afmt.SampleEncodingInt
	case LPCMInt16:
		format.BitDepth = 16
		format.Encoding = afmt.SampleEncodingInt
	case LPCMInt24:
		format.BitDepth = 24
		format.Encoding = afmt.SampleEncodingInt
	case LPCMInt32:
		format.BitDepth = 32
		format.Encoding = afmt.SampleEncodingInt
	case LPCMFloat32:
		format.BitDepth = 32
		format.Encoding = afmt.SampleEncodingFloat
	case LPCMFloat64:
		format.BitDepth = 64
		format.Encoding = afmt.SampleEncodingFloat
	case Alaw:
		format.BitDepth = 8
		format.Encoding = afmt.SampleEncodingUnknown
	default:
		panic(fmt.Errorf("au: unsupported encoding %d", e.encoding))
	}
	return format
}

func (e *Encoder) writeHeader() error {
	// Write magic
	if _, err := e.w.Write([]byte(magic)); err != nil {
		return err
	}

	// Write offset
	if err := binary.Write(e.w, binary.BigEndian, uint32(24+len(e.extraData))); err != nil {
		return err
	}

	// Write data size (placeholder)
	if err := binary.Write(e.w, binary.BigEndian, uint32(0xFFFFFFFF)); err != nil {
		return err
	}

	// Write encoding
	if err := binary.Write(e.w, binary.BigEndian, e.encoding); err != nil {
		return err
	}

	// Write sample rate
	if err := binary.Write(e.w, binary.BigEndian, uint32(e.format.SampleRate.Hertz())); err != nil {
		return err
	}

	// Write number of channels
	if err := binary.Write(e.w, binary.BigEndian, uint32(e.format.NumChannels)); err != nil {
		return err
	}

	// Write extra data
	if len(e.extraData) > 0 {
		if _, err := e.w.Write(e.extraData); err != nil {
			return err
		}
	}

	return nil
}

// WriteSamples encodes and writes samples.
func (e *Encoder) WriteSamples(p []float32) (int, error) {
	n, err := e.enc.WriteSamples(p)
	e.dataWritten += n
	return n, err
}

// Close finalizes the encoding process and writes the data size.
//
// It will NOT close the underlying writer, even if it implements [io.Closer].
// Closing the underlying writer is the owner's responsibility.
func (e *Encoder) Close() error {
	_, err := e.w.Seek(8, io.SeekStart)
	if err != nil {
		return fmt.Errorf("au: failed to seek to data size: %w", err)
	}

	if err := binary.Write(e.w, binary.BigEndian, uint32(e.dataWritten)); err != nil {
		return fmt.Errorf("au: failed to write data size: %w", err)
	}

	if _, err := e.w.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("au: failed to seek to end: %w", err)
	}

	return nil
}
