package g711

import (
	"io"

	"github.com/SladkyCitron/resona/aio"
)

// EncodeAlaw encodes a slice of float32 samples into A-law.
func EncodeAlaw(s []float32) []byte {
	b := make([]byte, len(s))
	for i := range s {
		i16 := int16(s[i] * (1<<15 - 1))
		if i16 >= 0 {
			b[i] = alawEnc[i16>>4]
		} else {
			b[i] = 0x7F & alawEnc[-i16>>4]
		}
	}
	return b
}

type alawEncoder struct {
	w io.Writer
}

// NewAlawEncoder returns an aio.SampleWriter that encodes and writes A-law samples to the provided [io.Writer].
func NewAlawEncoder(w io.Writer) aio.SampleWriter {
	return &alawEncoder{w: w}
}

func (e *alawEncoder) WriteSamples(p []float32) (int, error) {
	alaw := EncodeAlaw(p)
	return e.w.Write(alaw)
}

// DecodeAlaw decodes A-law encoded samples.
func DecodeAlaw(b []byte) []float32 {
	s := make([]float32, len(b))
	for i := range b {
		i16 := alawDec[b[i]]
		s[i] = float32(i16) / (1<<15 - 1)
	}
	return s
}

type alawDecoder struct {
	r   io.Reader
	buf []byte
}

// NewAlawDecoder returns an aio.SampleReader that reads and decodes A-law encoded samples from the provided [io.Reader].
func NewAlawDecoder(r io.Reader) aio.SampleReader {
	return &alawDecoder{r: r}
}

func (d *alawDecoder) ReadSamples(p []float32) (int, error) {
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

	f64buf := DecodeAlaw(d.buf[:n])
	copy(p, f64buf)

	return n, err
}
