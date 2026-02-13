package au

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/codec"
	"github.com/SladkyCitron/resona/encoding/g711"
	"github.com/SladkyCitron/resona/encoding/pcm"
	"github.com/SladkyCitron/resona/freq"
)

// Decoder represents the decoder for the AU file format.
// It implements codec.Decoder.
type Decoder struct {
	r        io.Reader
	dataRead int

	dec aio.SampleReader

	dataSize    uint32
	Encoding    uint32 // Encoding is the audio encoding type.
	sampleRate  uint32
	numChannels uint32
}

// NewDecoder creates a new [Decoder] and decodes the headers.
func NewDecoder(r io.Reader) (codec.Decoder, error) {
	d := &Decoder{r: r}

	// Read magic
	var buf [len(magic)]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, err
	}
	d.dataRead += len(buf)
	if string(buf[:]) != magic {
		return nil, errors.New("au: invalid header")
	}

	// Read offset
	var offset uint32
	if err := binary.Read(r, binary.BigEndian, &offset); err != nil {
		return nil, fmt.Errorf("au: failed to read offset: %w", err)
	}
	d.dataRead += 4

	// Read data size
	if err := binary.Read(r, binary.BigEndian, &d.dataSize); err != nil {
		return nil, fmt.Errorf("au: failed to read data size: %w", err)
	}
	d.dataRead += 4

	// Read encoding
	if err := binary.Read(r, binary.BigEndian, &d.Encoding); err != nil {
		return nil, fmt.Errorf("au: failed to read encoding: %w", err)
	}
	d.dataRead += 4

	if d.Encoding < 1 || (d.Encoding > 7 && d.Encoding != 27) {
		return nil, fmt.Errorf("au: unsupported encoding %d", d.Encoding)
	}

	// Read sample rate
	if err := binary.Read(r, binary.BigEndian, &d.sampleRate); err != nil {
		return nil, fmt.Errorf("au: failed to read sample rate: %w", err)
	}
	d.dataRead += 4

	// Read number of channels
	if err := binary.Read(r, binary.BigEndian, &d.numChannels); err != nil {
		return nil, fmt.Errorf("au: failed to read number of channels: %w", err)
	}
	d.dataRead += 4

	if _, err := io.CopyN(io.Discard, r, int64(offset)-int64(d.dataRead)); err != nil {
		return nil, fmt.Errorf("au: failed to skip to data: %w", err)
	}
	d.dataRead += int(offset) - d.dataRead

	switch d.Encoding {
	case Ulaw:
		d.dec = g711.NewUlawDecoder(r)
	case Alaw:
		d.dec = g711.NewAlawDecoder(r)
	default:
		d.dec = pcm.NewDecoder(r, d.SampleFormat())
	}
	d.dataRead = 0

	return d, nil
}

// Format returns the audio stream format.
func (d *Decoder) Format() afmt.Format {
	return afmt.Format{
		SampleRate:  freq.Frequency(d.sampleRate) * freq.Hertz,
		NumChannels: int(d.numChannels),
	}
}

// SampleFormat returns the sample format.
func (d *Decoder) SampleFormat() afmt.SampleFormat {
	format := afmt.SampleFormat{Endian: binary.BigEndian}
	switch d.Encoding {
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
		panic(fmt.Errorf("au: unsupported encoding %d", d.Encoding))
	}
	return format
}

// Len returns the total number of frames.
func (d *Decoder) Len() int {
	return int(d.dataSize) / d.SampleFormat().BitDepth
}

// ReadSamples reads float32 samples into p.
// It returns the number of samples read and/or an error.
func (d *Decoder) ReadSamples(p []float32) (int, error) {
	n, err := d.dec.ReadSamples(p)
	d.dataRead += n * (d.SampleFormat().BitDepth / 8)
	return n, err
}

// Seek seeks to the specified frame.
// It returns the new offset relative to the start and/or an error.
// It will return an error if the source is not an [io.Seeker].
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	s, ok := d.r.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("au: resource does not support seeking")
	}

	// Special case
	if offset == 0 && whence == io.SeekCurrent {
		return int64(d.dataRead) / int64(d.SampleFormat().BytesPerFrame(int(d.numChannels))), nil
	}

	frameSize := d.SampleFormat().BytesPerFrame(int(d.numChannels))
	totalFrames := int64(d.dataSize) / int64(frameSize)

	var target int64
	switch whence {
	case io.SeekStart:
		target = offset
	case io.SeekCurrent:
		target = int64(d.dataRead)/int64(frameSize) + offset
	case io.SeekEnd:
		target = int64(totalFrames) + offset
	default:
		return 0, fmt.Errorf("au: invalid seek whence")
	}

	if target < 0 || target > totalFrames {
		return 0, fmt.Errorf("au: seek out of bounds")
	}

	byteOffset := target * int64(frameSize)

	_, err := s.Seek(byteOffset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("au: failed to seek: %w", err)
	}

	d.dataRead = int(byteOffset)
	return target, nil
}

func init() {
	codec.RegisterFormat("au", magic, NewDecoder)
}
