package pcm

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
)

type decoder struct {
	r            io.Reader
	sampleFormat afmt.SampleFormat
	buf          []byte
}

// NewDecoder returns an aio.SampleReader that reads and decodes PCM samples from the provided [io.Reader].
func NewDecoder(r io.Reader, sampleFormat afmt.SampleFormat) aio.SampleReader {
	if sampleFormat.Endian == nil {
		sampleFormat.Endian = binary.NativeEndian
	}

	return &decoder{
		r:            r,
		sampleFormat: sampleFormat,
	}
}

func (d *decoder) ReadSamples(p []float32) (int, error) {
	if d.sampleFormat.BitDepth <= 0 {
		return 0, ErrInvalidBitDepth
	}
	if d.sampleFormat.Encoding <= 0 {
		return 0, ErrInvalidSampleEncoding
	}

	sampleSize := d.sampleFormat.BytesPerSample()

	numSamples := len(p)
	numBytes := numSamples * sampleSize
	if cap(d.buf) < numBytes {
		d.buf = make([]byte, numBytes)
	} else {
		d.buf = d.buf[:numBytes]
	}
	n, err := io.ReadFull(d.r, d.buf)
	switch err {
	case nil:
		// do nothing
	case io.ErrUnexpectedEOF:
		// decode the bytes that were read
	case io.EOF:
		return 0, io.EOF
	default:
		return 0, err
	}

	for i := range p[:n/sampleSize] {
		offset := i * sampleSize
		switch d.sampleFormat.Encoding {
		case afmt.SampleEncodingInt:
			switch d.sampleFormat.BitDepth {
			case 8:
				v := d.buf[offset]
				p[i] = float32(int8(v)) / (1<<7 - 1)
			case 16:
				v := int16(d.sampleFormat.Endian.Uint16(d.buf[offset:]))
				p[i] = float32(v) / (1<<15 - 1)
			case 24:
				b := d.buf[offset : offset+3]
				v := int32(uint24(b, d.sampleFormat.Endian))
				if v&(1<<23) != 0 {
					v |= ^0xFFFFFF
				}
				p[i] = float32(v) / (1<<23 - 1)
			case 32:
				v := int32(d.sampleFormat.Endian.Uint32(d.buf[offset:]))
				p[i] = float32(v) / (1<<31 - 1)
				/*
					case 64:
						v := int64(d.sampleFormat.Endian.Uint64(d.buf[offset:]))
						p[i] = float32(v) / (1<<63 - 1)
				*/
			default:
				return 0, ErrInvalidBitDepth
			}
		case afmt.SampleEncodingUint:
			switch d.sampleFormat.BitDepth {
			case 8:
				v := d.buf[offset]
				p[i] = float32(v)/127.5 - 1.0
				/*
					case 16:
						v := d.sampleFormat.Endian.Uint16(d.buf[offset:])
						p[i] = float32(v) / (1<<16 - 1)
					case 24:
						if offset+3 > len(d.buf) {
							return i, io.ErrUnexpectedEOF
						}
						b := d.buf[offset : offset+3]
						v := uint24(b, d.sampleFormat.Endian)
						p[i] = float32(v) / (1<<24 - 1)
					case 32:
						v := d.sampleFormat.Endian.Uint32(d.buf[offset:])
						p[i] = float32(v) / (1<<32 - 1)
					case 64:
						v := d.sampleFormat.Endian.Uint64(d.buf[offset:])
						p[i] = float32(v) / (1<<64 - 1)
				*/
			default:
				return 0, ErrInvalidBitDepth
			}
		case afmt.SampleEncodingFloat:
			switch d.sampleFormat.BitDepth {
			case 32:
				bits := d.sampleFormat.Endian.Uint32(d.buf[offset:])
				p[i] = math.Float32frombits(bits)
			case 64:
				bits := d.sampleFormat.Endian.Uint64(d.buf[offset:])
				p[i] = float32(math.Float64frombits(bits))
			default:
				return 0, ErrInvalidBitDepth
			}
		default:
			return 0, ErrInvalidSampleEncoding
		}
	}

	samplesRead := n / sampleSize
	if samplesRead > 0 {
		return samplesRead, nil
	}
	return 0, io.EOF
}

func uint24(p []byte, endian binary.ByteOrder) uint32 {
	if len(p) < 3 {
		return 0
	}
	switch endian {
	case binary.BigEndian:
		return uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
	case binary.LittleEndian:
		return uint32(p[2])<<16 | uint32(p[1])<<8 | uint32(p[0])
	default:
		panic("unsupported byte order")
	}
}

// Decode decodes the PCM byte slice into a slice of float32 samples.
func Decode(b []byte, sampleFormat afmt.SampleFormat) ([]float32, error) {
	dec := NewDecoder(bytes.NewReader(b), sampleFormat)
	p := make([]float32, len(b)/sampleFormat.BytesPerSample())
	n, err := dec.ReadSamples(p)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return p[:n], nil
}
