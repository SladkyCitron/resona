package audio

import (
	"errors"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
)

// A Reader implements the afmt.Formatter, aio.SampleReader, aio.SampleReaderAt, aio.SampleWriterTo,
// [io.Seeker] and aio.SingleSampleScanner interfaces by reading from a float32 sample slice.
// Unlike a [Buffer], a Reader is read-only, does not support writing and supports seeking.
// The zero value for Reader operates like a Reader of an empty slice i.e. no samples.
// Using [Reader.Fmt] and [Reader.Format] is optional, but recommended.
type Reader struct {
	Fmt afmt.Format
	s   []float32
	i   int64
}

// Len returns the number of samples of the unread portion of the slice.
func (r *Reader) Len() int {
	if r.i >= int64(len(r.s)) {
		return 0
	}
	return int(int64(len(r.s)) - r.i)
}

// Size returns the original length of the underlying sample slice.
// Size is the number of samples available for reading via [Reader.ReadSamplesAt].
// The result is unaffected by any method calls except [Reader.Reset].
func (r *Reader) Size() int64 { return int64(len(r.s)) }

// ReadSamples implements the aio.SampleReader interface.
func (r *Reader) ReadSamples(p []float32) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += int64(n)
	return
}

// ReadSamplesAt implements the aio.SampleReaderAt interface.
func (r *Reader) ReadSamplesAt(p []float32, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("audio.Reader.ReadSamplesAt: negative offset")
	}
	if off >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(p, r.s[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}

// Seek implements the [io.Seeker] interface.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.i + offset
	case io.SeekEnd:
		abs = int64(len(r.s)) + offset
	default:
		return 0, errors.New("audio.Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("audio.Reader.Seek: negative position")
	}
	r.i = abs
	return abs, nil
}

// WriteSamplesTo implements the aio.SampleWriterTo interface.
func (r *Reader) WriteSamplesTo(w aio.SampleWriter) (n int64, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, nil
	}
	s := r.s[r.i:]
	m, err := w.WriteSamples(s)
	if m > len(s) {
		panic("audio.Reader.WriteSamplesTo: invalid WriteSamples count")
	}
	r.i += int64(m)
	n = int64(m)
	if m != len(s) && err == nil {
		err = io.ErrShortWrite
	}
	return
}

// Format implements afmt.Formatter.
func (r *Reader) Format() afmt.Format {
	return r.Fmt
}

// Reset resets the [Reader] to be reading from p.
func (r *Reader) Reset(p []float32) { *r = Reader{r.Fmt, p, 0} }

// NewReader returns a new [Reader] reading from p.
func NewReader(p []float32) *Reader { return &Reader{afmt.Format{}, p, 0} }
