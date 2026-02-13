package audio

import (
	"io"

	"github.com/SladkyCitron/resona/aio"
)

// Upmixer duplicates samples and converts mono to multiple-channel audio (e.g. stereo, 5.1 surround).
type Upmixer struct {
	src      aio.SampleReader
	numChans int
	monoBuf  []float32
}

// NewUpmixer creates a new [Upmixer] with the specified target number of channels to upmix to.
func NewUpmixer(r aio.SampleReader, numChans int) *Upmixer {
	return &Upmixer{
		src:      r,
		numChans: numChans,
	}
}

func (u *Upmixer) ReadSamples(p []float32) (int, error) {
	if len(p)%u.numChans != 0 {
		return 0, io.ErrShortBuffer
	}

	monoSamples := len(p) / u.numChans

	if cap(u.monoBuf) < monoSamples {
		u.monoBuf = make([]float32, monoSamples)
	} else {
		u.monoBuf = u.monoBuf[:monoSamples]
	}

	n, err := u.src.ReadSamples(u.monoBuf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	for i := range u.monoBuf {
		for ch := range u.numChans {
			p[i*u.numChans+ch] = u.monoBuf[i]
		}
	}

	return n * u.numChans, err
}
