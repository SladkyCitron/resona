package effect

import "github.com/SladkyCitron/resona/dsp/filter"

// Filter wraps a filter.Filter and filters the audio signal using it.
type Filter struct {
	f filter.Filter
}

// NewFilter creates a new [Filter].
func NewFilter(f filter.Filter) *Filter {
	return &Filter{f: f}
}

func (f *Filter) Process(p []float32) error {
	for i := range p {
		p[i] = f.f.ProcessSingle(p[i])
	}
	return nil
}
