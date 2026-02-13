// Package ffplay provides a FFplay-based playback driver.
//
// Note: FFplay buffers several seconds of audio before playback actually starts.
// This causes playback position reports to be offset until FFplay begins output.
package ffplay

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/dsp"
	"github.com/SladkyCitron/resona/playback"
)

// Driver represents the driver.
type Driver struct {
	cancel context.CancelFunc
}

func getChanLayout(numCh int) string {
	switch numCh {
	case 1:
		return "mono"
	case 2:
		return "stereo"
	case 3:
		return "2.1"
	case 4:
		return "4.0"
	case 5:
		return "5.0"
	case 6:
		return "5.1"
	case 7:
		return "6.1"
	case 8:
		return "7.1"
	default:
		// Fall back to an explicit layout string
		return fmt.Sprintf("FL+FR+%dC", numCh-2)
	}
}

// Init initializes the driver based on the format and source.
// It blocks until the driver is ready.
func (d *Driver) Init(format afmt.Format, src aio.SampleReader) error {
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	cmd := exec.CommandContext(
		ctx,
		"ffplay",
		"-hide_banner",
		"-loglevel", "panic",
		"-vn", "-nodisp", // no video
		"-volume", "100",
		"-f", "f32le", // float32 little-endian
		"-ar", fmt.Sprintf("%.0f", format.SampleRate.Hertz()),
		"-ch_layout", getChanLayout(format.NumChannels),
		"-i", "pipe:0",
	)
	cmd.Stderr = os.Stderr
	cmd.Stdin = &pcmReader{src: src}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("ffplay: starting ffplay command: %w", err)
	}

	return nil
}

func (d *Driver) Close() error {
	d.cancel()
	return nil
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
	playback.Register("ffplay", &Driver{})
}
