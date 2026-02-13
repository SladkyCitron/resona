package pcm

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/dsp"
)

type ebitenS16 struct {
	src aio.SampleReader
	buf []float32
}

func (e *ebitenS16) Read(p []byte) (int, error) {
	const sampleSize = 2 // int16 size = 2 bytes

	numSamples := len(p) / sampleSize

	if cap(e.buf) < numSamples {
		e.buf = make([]float32, numSamples)
	} else {
		e.buf = e.buf[:numSamples]
	}
	clear(e.buf)

	n, err := e.src.ReadSamples(e.buf)
	if err != nil && n == 0 {
		return 0, err
	}
	for i := range e.buf[:n] {
		v := int16(dsp.Clamp(e.buf[i]) * (1<<15 - 1))
		binary.LittleEndian.PutUint16(p[i*sampleSize:], uint16(v))
	}
	return n * sampleSize, err
}

// Ebiten seeking is different from Resona seeking

func (e *ebitenS16) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := e.src.(io.Seeker)
	if !ok {
		return 0, errors.New("pcm ebitenS16: resource is not seekable")
	}

	const sampleSize = 2 // int16 size = 2 bytes
	offset /= sampleSize

	return seeker.Seek(offset, whence)
}

// NewEbitenS16Encoder creates an [io.Reader] that encodes samples from
// the given aio.SampleReader to 16-bit signed little-endian linear PCM format
// for use with Ebiten's audio APIs.
func NewEbitenS16Encoder(src aio.SampleReader) io.Reader {
	return &ebitenS16{src: src}
}

type ebitenF32 struct {
	src aio.SampleReader
	buf []float32
}

func (e *ebitenF32) Read(p []byte) (int, error) {
	const sampleSize = 4 // float32 size = 4 bytes

	numSamples := len(p) / sampleSize

	if cap(e.buf) < numSamples {
		e.buf = make([]float32, numSamples)
	} else {
		e.buf = e.buf[:numSamples]
	}
	clear(e.buf)

	n, err := e.src.ReadSamples(e.buf)
	if err != nil && n == 0 {
		return 0, err
	}
	for i := range e.buf[:n] {
		binary.LittleEndian.PutUint32(p[i*sampleSize:], math.Float32bits(dsp.Clamp(e.buf[i])))
	}
	return n * sampleSize, err
}

func (e *ebitenF32) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := e.src.(io.Seeker)
	if !ok {
		return 0, errors.New("pcm ebitenF32: resource is not seekable")
	}

	const sampleSize = 4 // float32 size = 4 bytes
	offset /= sampleSize

	return seeker.Seek(offset, whence)
}

// NewEbitenF32Encoder creates an [io.Reader] that encodes samples from
// the given aio.SampleReader to 32-bit float little-endian linear PCM format
// for use with Ebiten's F32 audio APIs.
func NewEbitenF32Encoder(src aio.SampleReader) io.Reader {
	return &ebitenF32{src: src}
}
