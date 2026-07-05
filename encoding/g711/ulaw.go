package g711

import (
	"io"

	"github.com/SladkyCitron/resona/aio"
)

// EncodeUlaw encodes a slice of float32 samples into μ-law.
func EncodeUlaw(s []float32) []byte {
	b := make([]byte, len(s))
	for i := range s {
		i16 := int16(s[i] * (1<<15 - 1))
		if i16 >= 0 {
			b[i] = ulawEnc[i16>>4]
		} else {
			b[i] = 0x7F & ulawEnc[-i16>>4]
		}
	}
	return b
}

type ulawEncoder struct {
	w io.Writer
}

// NewUlawEncoder returns an aio.SampleWriter that encodes and writes μ-law samples to the provided [io.Writer].
func NewUlawEncoder(w io.Writer) aio.SampleWriter {
	return &ulawEncoder{w: w}
}

func (e *ulawEncoder) WriteSamples(p []float32) (int, error) {
	ulaw := EncodeUlaw(p)
	return e.w.Write(ulaw)
}

// DecodeUlaw decodes μ-law encoded samples.
func DecodeUlaw(b []byte) []float32 {
	s := make([]float32, len(b))
	for i := range b {
		i16 := ulawDec[b[i]]
		s[i] = float32(i16) / (1<<15 - 1)
	}
	return s
}

type ulawDecoder struct {
	r   io.Reader
	buf []byte
}

// NewUlawDecoder returns an aio.SampleReader that reads and decodes μ-law encoded samples from the provided [io.Reader].
func NewUlawDecoder(r io.Reader) aio.SampleReader {
	return &ulawDecoder{r: r}
}

func (d *ulawDecoder) ReadSamples(p []float32) (int, error) {
	numSamples := len(p)
	if cap(d.buf) < numSamples {
		d.buf = make([]byte, numSamples)
	} else {
		d.buf = d.buf[:numSamples]
	}

	n, err := d.r.Read(d.buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	f32buf := DecodeUlaw(d.buf[:n])
	copy(p, f32buf)

	return n, err
}
