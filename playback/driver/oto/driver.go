// Package oto provides a cross-platform [Oto]-based playback driver.
//
// Note: Oto reads PCM in ~500ms chunks depending on the platform backend.
// Position reporting may appear quantized during playback.
//
// [Oto]: https://github.com/ebitengine/oto
package oto

import (
	"encoding/binary"
	"math"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/dsp"
	"github.com/SladkyCitron/resona/playback"
	"github.com/ebitengine/oto/v3"
)

// Driver represents the driver.
type Driver struct {
	ctx    *oto.Context
	player *oto.Player
}

// Init initializes the driver based on the format and source.
// It blocks until the driver is ready.
func (d *Driver) Init(format afmt.Format, src aio.SampleReader) error {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   int(format.SampleRate.Hertz()),
		ChannelCount: format.NumChannels,
		Format:       oto.FormatFloat32LE,
	})
	if err != nil {
		return err
	}
	<-ready

	d.ctx = ctx

	d.player = ctx.NewPlayer(&pcmReader{src: src})
	d.player.Play()
	return nil
}

// Close closes audio playback.
// However, the underlying driver keeps existing until the process dies,
// as closing it is not supported (see [Oto issue #149]).
//
// In most cases, there is no need to call Close even when the program doesn't play
// audio anymore, because the driver closes when the process dies.
//
// [Oto issue #149]: https://github.com/ebitengine/oto/issues/149
func (d *Driver) Close() error {
	if err := d.player.Close(); err != nil {
		return err
	}
	return d.ctx.Suspend()
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

func init() {
	playback.Register("oto", &Driver{}) // register driver
}
