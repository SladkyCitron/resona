package audio

import (
	"io"

	"github.com/SladkyCitron/resona/aio"
)

type Downmixer struct {
	src         aio.SampleReader
	srcNumChans int
	buf         []float32
}

// NewDownmixer creates a new [Downmixer] that converts multi-channel audio (e.g. stereo, 5.1 surround) to mono by averaging.
func NewDownmixer(r aio.SampleReader, srcNumChans int) *Downmixer {
	return &Downmixer{
		src:         r,
		srcNumChans: srcNumChans,
	}
}

func (d *Downmixer) ReadSamples(p []float32) (int, error) {
	multiSamples := len(p) * d.srcNumChans

	if cap(d.buf) < multiSamples {
		d.buf = make([]float32, multiSamples)
	} else {
		d.buf = d.buf[:multiSamples]
	}

	n, err := d.src.ReadSamples(d.buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	if n%d.srcNumChans != 0 {
		return 0, io.ErrUnexpectedEOF
	}

	for i := range n / d.srcNumChans {
		var sum float32
		for ch := range d.srcNumChans {
			sum += d.buf[i*d.srcNumChans+ch]
		}
		p[i] = sum / float32(d.srcNumChans)
	}

	return n / d.srcNumChans, err
}
