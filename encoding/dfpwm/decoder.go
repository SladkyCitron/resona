package dfpwm

import (
	"bytes"
	"io"

	"github.com/SladkyCitron/resona/aio"
)

// Original C implementation: https://github.com/ChenThread/dfpwm/blob/master/1a/audecmp.c
/*
DFPWM1a (Dynamic Filter Pulse Width Modulation) codec - C Implementation
by Ben "GreaseMonkey" Russell, 2012, 2016
Public Domain

Decompression Component
*/

const PostFilt = 140

type decoder struct {
	r   io.Reader
	buf []byte

	q  int
	s  int
	lt int
	fq int
	fs int
}

// NewDecoder returns an aio.SampleReader that reads and decodes DFPWM encoded samples from the provided [io.Reader].
func NewDecoder(r io.Reader) aio.SampleReader {
	return &decoder{
		r:  r,
		q:  0,
		s:  0,
		lt: -128,
		fq: 0,
		fs: PostFilt,
	}
}

func (dec *decoder) ReadSamples(p []float32) (int, error) {
	lenCompressed := len(p) / 8
	if cap(dec.buf) < lenCompressed {
		dec.buf = make([]byte, lenCompressed)
	} else {
		dec.buf = dec.buf[:lenCompressed]
	}

	n, err := dec.r.Read(dec.buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	for i := range dec.buf[:n] {
		// get bits
		d := dec.buf[i]

		idx := i * 8

		for j := range 8 {
			// set target
			var t int
			if d&1 == 1 {
				t = 127
			} else {
				t = -128
			}
			d >>= 1

			// adjust charge
			var nq int = dec.q + ((dec.s*(t-dec.q) + (1 << (Prec - 1))) >> Prec)
			if nq == dec.q && nq != t {
				if t == 127 {
					dec.q++
				} else {
					dec.q--
				}
			}
			lq := dec.q
			dec.q = nq

			// adjust strength
			var st int
			if t != dec.lt {
				st = 0
			} else {
				st = (1 << Prec) - 1
			}
			ns := dec.s
			if ns != st {
				if st != 0 {
					ns++
				} else {
					ns--
				}
			}
			if Prec > 8 && ns < 1+(1<<(Prec-8)) {
				ns = 1 + (1 << (Prec - 8))
			}
			dec.s = ns

			// FILTER: perform antijerk
			var ov int
			if t != dec.lt {
				ov = (nq + lq) >> 1
			} else {
				ov = nq
			}

			// FILTER: perform LPF
			dec.fq += ((dec.fs*(ov-dec.fq) + 0x80) >> 8)
			ov = dec.fq

			// convert int8 => float32
			p[idx+j] = float32(ov) / 128.0

			dec.lt = t
		}
	}
	return n * 8, err
}

// Decode decodes the DFPWM byte slice into a slice of float64 samples.
func Decode(b []byte) ([]float32, error) {
	dec := NewDecoder(bytes.NewReader(b))
	p := make([]float32, len(b)*8)
	n, err := dec.ReadSamples(p)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return p[:n], nil
}

// DecodedLen returns the length of a decoding of x source bytes. Specifically, it returns x * 8.
func DecodedLen(x int) int {
	return x * 8
}
