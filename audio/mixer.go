package audio

import (
	"io"

	"github.com/SladkyCitron/resona/aio"
)

// Mixer allows for dynamic mixing of arbitrary number of SampleReaders.
//
// Mixer automatically removes drained SampleReaders. Depending on [Mixer.KeepAlive],
// Mixer will either output silence or drain when all SampleReaders have been drained.
// By default, it will output silence.
//
// The zero value for Mixer is an empty mixer ready to use.
type Mixer struct {
	readers       []aio.SampleReader
	stopWhenEmpty bool
}

// NewMixer creates a new [Mixer] using readers as its initial readers.
//
// In most cases, new([Mixer]) (or just declaring a [Mixer] variable) is sufficient
// to create a new [Mixer].
func NewMixer(readers ...aio.SampleReader) *Mixer {
	return &Mixer{
		readers:       readers,
		stopWhenEmpty: false,
	}
}

// KeepAlive sets the [Mixer] whether to keep playing silence when all readers have drained (true),
// or stop playing and return an [io.EOF] (false).
func (m *Mixer) KeepAlive(KeepAlive bool) {
	m.stopWhenEmpty = !KeepAlive
}

// Len returns the number of readers currently playing in the [Mixer].
func (m *Mixer) Len() int {
	return len(m.readers)
}

// Add adds new reader(s) to the [Mixer].
func (m *Mixer) Add(readers ...aio.SampleReader) {
	m.readers = append(m.readers, readers...)
}

// Clear wipes and removes all readers from the [Mixer].
func (m *Mixer) Clear() {
	clear(m.readers)
}

//TODO: improve

// ReadSamples reads the samples of all readers currently playing in the [Mixer], mixed together.
// Depending on [Mixer.KeepAlive], this will either output silence or drain and return an [io.EOF].
func (m *Mixer) ReadSamples(p []float32) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	// If there are no readers
	if len(m.readers) == 0 {
		if m.stopWhenEmpty {
			return 0, io.EOF
		}
		// Output silence
		clear(p)
		return len(p), nil
	}

	var (
		buf     = make([]float32, len(p))
		maxRead int
		keep    []aio.SampleReader
		anyRead bool
		readErr error
	)

	for _, r := range m.readers {
		if r == nil {
			continue
		}
		n, err := r.ReadSamples(buf)
		if n > 0 {
			anyRead = true
			for i := 0; i < n && i < len(p); i++ {
				p[i] += buf[i]
			}
			if n > maxRead {
				maxRead = n
			}
		}

		if err == nil || (err == io.EOF && n > 0) {
			keep = append(keep, r)
		}

		// Keep first non-EOF error
		if err != nil && err != io.EOF && readErr == nil {
			readErr = err
		}
	}

	m.readers = keep

	if maxRead == 0 && !anyRead {
		if m.stopWhenEmpty && len(m.readers) == 0 {
			return 0, io.EOF
		}
		clear(p)
		return len(p), nil
	}

	return maxRead, readErr
}
