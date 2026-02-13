package pcm

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/dsp"
)

type encoder struct {
	w            io.Writer
	sampleFormat afmt.SampleFormat
	buf          []byte
}

// NewEncoder returns an aio.SampleWriter that encodes and writes PCM samples to the provided [io.Writer].
func NewEncoder(w io.Writer, sampleFormat afmt.SampleFormat) aio.SampleWriter {
	if sampleFormat.Endian == nil {
		sampleFormat.Endian = binary.NativeEndian
	}

	return &encoder{
		w:            w,
		sampleFormat: sampleFormat,
	}
}

func (e *encoder) WriteSamples(p []float32) (int, error) {
	if e.sampleFormat.BitDepth <= 0 {
		return 0, ErrInvalidBitDepth
	}
	if e.sampleFormat.Encoding <= 0 {
		return 0, ErrInvalidSampleEncoding
	}

	sampleSize := e.sampleFormat.BytesPerSample()
	totalBytes := len(p) * sampleSize

	if cap(e.buf) < totalBytes {
		e.buf = make([]byte, totalBytes)
	} else {
		e.buf = e.buf[:totalBytes]
	}

	for i := range p {
		s := float64(dsp.Clamp(p[i]))
		offset := i * sampleSize

		switch e.sampleFormat.Encoding {
		case afmt.SampleEncodingInt:
			switch e.sampleFormat.BitDepth {
			case 8:
				e.buf[offset] = byte(int8(s * 127))
			case 16:
				v := int16(s * (1<<15 - 1))
				e.sampleFormat.Endian.PutUint16(e.buf[offset:], uint16(v))
			case 24:
				v := int32(s * (1<<23 - 1))
				putUint24(e.buf[offset:], uint32(v), e.sampleFormat.Endian)
			case 32:
				v := int32(s * (1<<31 - 1))
				e.sampleFormat.Endian.PutUint32(e.buf[offset:], uint32(v))
			default:
				return 0, ErrInvalidBitDepth
			}
		case afmt.SampleEncodingUint:
			switch e.sampleFormat.BitDepth {
			case 8:
				v := byte((s + 1.0) * 0.5 * 255)
				e.buf[offset] = v
			default:
				return 0, ErrInvalidBitDepth
			}
		case afmt.SampleEncodingFloat:
			switch e.sampleFormat.BitDepth {
			case 32:
				e.sampleFormat.Endian.PutUint32(e.buf[offset:], math.Float32bits(float32(s)))
			case 64:
				e.sampleFormat.Endian.PutUint64(e.buf[offset:], math.Float64bits(s))
			default:
				return 0, ErrInvalidBitDepth
			}
		default:
			return 0, ErrInvalidSampleEncoding
		}
	}

	n, err := e.w.Write(e.buf)
	if err != nil {
		return 0, err
	}

	return n / sampleSize, nil
}

func putUint24(p []byte, v uint32, endian binary.ByteOrder) {
	if len(p) < 3 {
		return
	}
	switch endian {
	case binary.BigEndian:
		p[0] = byte(v >> 16)
		p[1] = byte(v >> 8)
		p[2] = byte(v)
	case binary.LittleEndian:
		p[0] = byte(v)
		p[1] = byte(v >> 8)
		p[2] = byte(v >> 16)
	default:
		panic("unsupported byte order")
	}
}

// Encode encodes a slice of float32 samples into a PCM byte slice.
func Encode(s []float32, sampleFormat afmt.SampleFormat) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf, sampleFormat)
	_, err := enc.WriteSamples(s)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
