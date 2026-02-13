// Package effect provides various audio effects and DSP chain components.
package effect

import (
	"io"

	"github.com/SladkyCitron/resona/aio"
)

// Effect represents an audio effect that can process audio samples.
type Effect interface {
	// Process applies the effect to the input sample and modifies it.
	// It may return an error if processing fails.
	Process(p []float32) error
}

// EffectFunc is an [Effect] created by simply wrapping a processing function.
type EffectFunc func(p []float32) error

// Process calls the wrapped processing function.
func (f EffectFunc) Process(p []float32) error {
	return f(p)
}

type reader struct {
	r   aio.SampleReader
	fx  Effect
	buf []float32
}

// Reader wraps an aio.SampleReader and applies the given [Effect] to its output.
func Reader(r aio.SampleReader, fx Effect) *reader {
	return &reader{r: r, fx: fx}
}

func (r *reader) ReadSamples(p []float32) (int, error) {
	if cap(r.buf) < len(p) {
		r.buf = make([]float32, len(p))
	} else {
		r.buf = r.buf[:len(p)]
	}

	n, err := r.r.ReadSamples(r.buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	if err := r.fx.Process(r.buf[:n]); err != nil {
		return 0, err
	}

	copy(p, r.buf)

	return n, err
}

// Apply applies the given [Effect] to the input sample slice and returns the processed ones.
// To apply the effect in-place, use [Effect.Process] directly instead.
func Apply(p []float32, fx Effect) ([]float32, error) {
	out := make([]float32, len(p))
	if err := fx.Process(p); err != nil {
		return nil, err
	}
	return out, nil
}
