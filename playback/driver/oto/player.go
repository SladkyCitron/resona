package oto

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/dsp"
	"github.com/SladkyCitron/resona/playback/driver"
	"github.com/ebitengine/oto/v3"
)

var _ driver.Player = (*player)(nil)

type player struct {
	p *oto.Player
}

func (d *Driver) NewPlayer(src aio.SampleReader) driver.Player {
	return &player{p: d.ctx.NewPlayer(&pcmReader{src: src})}
}

func (p *player) Play() {
	p.p.Play()
}

func (p *player) PlayWithDone() chan struct{} {
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for p.p.IsPlaying() {
			continue
		}
		close(done)
	}()

	p.Play()

	return done
}

// pcmReader is an [io.Reader] that wraps aio.SampleReader and encodes audio to float32 little endian PCM.
type pcmReader struct {
	src aio.SampleReader
	buf []float32
}

func (r *pcmReader) Read(p []byte) (int, error) {
	const sampleSize = 4 // float32 size = 4 bytes

	numSamples := len(p) / sampleSize

	if cap(r.buf) < numSamples {
		r.buf = make([]float32, numSamples)
	} else {
		r.buf = r.buf[:numSamples]
	}
	clear(r.buf)

	n, err := r.src.ReadSamples(r.buf)
	if err != nil && n == 0 {
		return 0, err
	}
	for i := range r.buf[:n] {
		binary.LittleEndian.PutUint32(p[i*sampleSize:], math.Float32bits(dsp.Clamp(r.buf[i])))
	}
	return n * sampleSize, err
}
