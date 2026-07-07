// Package oto provides a cross-platform [Oto]-based playback driver.
//
// Note: Oto reads PCM in ~500ms chunks depending on the platform backend.
// Position reporting may appear quantized during playback.
//
// [Oto]: https://github.com/ebitengine/oto
package oto

import (
	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/playback"
	"github.com/SladkyCitron/resona/playback/driver"
	"github.com/ebitengine/oto/v3"
)

var _ driver.Driver = (*Driver)(nil)

// Driver represents the driver.
type Driver struct {
	ctx *oto.Context
}

// Init initializes the driver based on the format and source.
// It blocks until the driver is ready.
func (d *Driver) Init(format afmt.Format, bufferSize int) error {
	op := &oto.NewContextOptions{
		SampleRate:   int(format.SampleRate.Hertz()),
		ChannelCount: format.NumChannels,
		Format:       oto.FormatFloat32LE,
	}
	if bufferSize != 0 {
		op.BufferSize = afmt.NumFramesToDuration(format.SampleRate, bufferSize)
	}

	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		return err
	}
	<-ready

	d.ctx = ctx
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
	return d.ctx.Suspend()
}

func init() {
	playback.Register("oto", &Driver{}) // register driver
}
