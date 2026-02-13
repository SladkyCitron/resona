package audio

/*
import (
	"sync/atomic"

	"github.com/SladkyCitron/resona/aio"
)

// Parameter represents a float32 control-rate parameter i.e. a "patch cable".
// It can be either a constant value, or can be mounted to another aio.SampleReader
// to allow modulation.
type Parameter struct {
	constValue atomic.Value
	src        aio.SampleReader
}

// NewParameter creates a new [Parameter] with the specified constant value.
func NewParameter(v float32) *Parameter {
	var av atomic.Value
	av.Store(v)

	return &Parameter{constValue: av}
}

// Get returns the current constant value (used when not modulated).
func (p *Parameter) Get() float32 {
	return p.constValue.Load().(float32)
}

// Set updates the constant value (safe if unmounted).
func (p *Parameter) Set(v float32) {
	p.constValue.Store(v)
}

// IsConst returns true if this parameter is not modulated.
func (p *Parameter) IsConst() bool {
	return p.src == nil
}

// Mount connects an audio-rate modulation source.
func (p *Parameter) Mount(src aio.SampleReader) {
	p.src = src
}

// Unmount removes any modulation source.
func (p *Parameter) Unmount() {
	p.src = nil
}

// ReadSamples fills buf with either the constant value or modulated samples.
func (p *Parameter) ReadSamples(buf []float32) (int, error) {
	if p.src == nil {
		for i := range buf {
			buf[i] = p.constValue.Load().(float32)
		}
		return len(buf), nil
	}
	return p.src.ReadSamples(buf)
}
*/
