// Portions of this code are derived from the Go standard library's bufio package.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package abufio implements buffered I/O for audio, similar to [bufio] but for audio.
// It wraps an aio.SampleReader or aio.SampleWriter object, creating another one that
// also implements the interface but provides buffering.
package abufio

import (
	"errors"
	"io"

	"github.com/SladkyCitron/resona/aio"
)

const defaultBufSize = 1024 // recommended for audio, I think

var (
	ErrInvalidUnreadSample = errors.New("bufio: invalid use of UnreadSample")
	ErrBufferFull          = errors.New("bufio: buffer full")
	ErrNegativeCount       = errors.New("bufio: negative count")
)

// Reader implements buffering for an aio.SampleReader object.
// A new Reader is created by calling [NewReader] or [NewReaderSize];
// alternatively the zero value of a Reader may be used after calling [Reset]
// on it.
type Reader struct {
	buf  []float32
	rd   aio.SampleReader // reader provided by the client
	r, w int              // buf read and write positions
	err  error
}

const minReadBufferSize = 16
const maxConsecutiveEmptyReads = 100

// NewReaderSize returns a new [Reader] whose buffer has at least the specified
// size. If the argument io.Reader is already a [Reader] with large enough
// size, it returns the underlying [Reader].
func NewReaderSize(rd aio.SampleReader, size int) *Reader {
	// Is it already a Reader?
	b, ok := rd.(*Reader)
	if ok && len(b.buf) >= size {
		return b
	}
	r := new(Reader)
	r.reset(make([]float32, max(size, minReadBufferSize)), rd)
	return r
}

// NewReader returns a new [Reader] whose buffer has the default size.
func NewReader(rd aio.SampleReader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

// Size returns the size of the underlying buffer in samples.
func (b *Reader) Size() int { return len(b.buf) }

// Reset discards any buffered audio, resets all state, and switches
// the buffered reader to read from r.
// Calling Reset on the zero value of [Reader] initializes the internal buffer
// to the default size.
// Calling b.Reset(b) (that is, resetting a [Reader] to itself) does nothing.
func (b *Reader) Reset(r aio.SampleReader) {
	if b == r {
		return
	}
	if b.buf == nil {
		b.buf = make([]float32, defaultBufSize)
	}
	b.reset(b.buf, r)
}

func (b *Reader) reset(buf []float32, r aio.SampleReader) {
	*b = Reader{
		buf: buf,
		rd:  r,
	}
}

var errNegativeRead = errors.New("bufio: reader returned negative count from Read")

// fill reads a new chunk into the buffer.
func (b *Reader) fill() {
	// Slide existing data to beginning.
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}

	if b.w >= len(b.buf) {
		panic("bufio: tried to fill full buffer")
	}

	// Read new data: try a limited number of times.
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := b.rd.ReadSamples(b.buf[b.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
		b.w += n
		if err != nil {
			b.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	b.err = io.ErrNoProgress
}

func (b *Reader) readErr() error {
	err := b.err
	b.err = nil
	return err
}

// Peek returns the next n samples without advancing the reader. The samples stop
// being valid at the next read call. If necessary, Peek will read more samples
// into the buffer in order to make n samples available. If Peek returns fewer
// than n samples, it also returns an error explaining why the read is short.
// The error is [ErrBufferFull] if n is larger than b's buffer size.
//
// Calling Peek prevents a [Reader.UnreadSample] call from succeeding
// until the next read operation.
func (b *Reader) Peek(n int) ([]float32, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}

	for b.w-b.r < n && b.w-b.r < len(b.buf) && b.err == nil {
		b.fill() // b.w-b.r < len(b.buf) => buffer is not full
	}

	if n > len(b.buf) {
		return b.buf[b.r:b.w], ErrBufferFull
	}

	// 0 <= n <= len(b.buf)
	var err error
	if avail := b.w - b.r; avail < n {
		// not enough data in buffer
		n = avail
		err = b.readErr()
		if err == nil {
			err = ErrBufferFull
		}
	}
	return b.buf[b.r : b.r+n], err
}

// Discard skips the next n samples, returning the number of samples discarded.
//
// If Discard skips fewer than n samples, it also returns an error.
// If 0 <= n <= b.Buffered(), Discard is guaranteed to succeed without
// reading from the underlying aio.SampleReader.
func (b *Reader) Discard(n int) (discarded int, err error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	if n == 0 {
		return
	}

	remain := n
	for {
		skip := b.Buffered()
		if skip == 0 {
			b.fill()
			skip = b.Buffered()
		}
		if skip > remain {
			skip = remain
		}
		b.r += skip
		remain -= skip
		if remain == 0 {
			return n, nil
		}
		if b.err != nil {
			return n - remain, b.readErr()
		}
	}
}

// ReadSamples reads data into p.
// It returns the number of samples read into p.
// The samples are taken from at most one read on the underlying [Reader],
// hence n may be less than len(p).
// To read exactly len(p) samples, use aio.ReadFull(b, p).
// If the underlying [Reader] can return a non-zero count with [io.EOF],
// then this ReadSamples method can do so as well; see the [aio.SampleReader] docs.
func (b *Reader) ReadSamples(p []float32) (n int, err error) {
	n = len(p)
	if n == 0 {
		if b.Buffered() > 0 {
			return 0, nil
		}
		return 0, b.readErr()
	}
	if b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		if len(p) >= len(b.buf) {
			// Large read, empty buffer.
			// Read directly into p to avoid copy.
			n, b.err = b.rd.ReadSamples(p)
			if n < 0 {
				panic(errNegativeRead)
			}
			return n, b.readErr()
		}
		// One read.
		// Do not use b.fill, which will loop.
		b.r = 0
		b.w = 0
		n, b.err = b.rd.ReadSamples(b.buf)
		if n < 0 {
			panic(errNegativeRead)
		}
		if n == 0 {
			return 0, b.readErr()
		}
		b.w += n
	}

	n = copy(p, b.buf[b.r:b.w])
	b.r += n
	return n, nil
}

// Buffered returns the number of samples that can be read from the current buffer.
func (b *Reader) Buffered() int { return b.w - b.r }

// WriteSamplesTo implements aio.SampleWriterTo.
// This may make multiple calls to the [Reader.ReadSamples] method of the underlying [Reader].
// If the underlying reader supports the [Reader.WriteSamplesTo] method,
// this calls the underlying [Reader.WriteSamplesTo] without buffering.
func (b *Reader) WriteSamplesTo(w aio.SampleWriter) (n int64, err error) {
	if b.r < b.w {
		n, err = b.writeBuf(w)
		if err != nil {
			return
		}
	}

	if r, ok := b.rd.(aio.SampleWriterTo); ok {
		m, err := r.WriteSamplesTo(w)
		n += m
		return n, err
	}

	if w, ok := w.(aio.SampleReaderFrom); ok {
		m, err := w.ReadSamplesFrom(b.rd)
		n += m
		return n, err
	}

	if b.w-b.r < len(b.buf) {
		b.fill() // buffer not full
	}

	for b.r < b.w {
		// b.r < b.w => buffer is not empty
		m, err := b.writeBuf(w)
		n += m
		if err != nil {
			return n, err
		}
		b.fill() // buffer is empty
	}

	if b.err == io.EOF {
		b.err = nil
	}

	return n, b.readErr()
}

var errNegativeWrite = errors.New("bufio: writer returned negative count from WriteSamples")

// writeBuf writes the [Reader]'s buffer to the writer.
func (b *Reader) writeBuf(w aio.SampleWriter) (int64, error) {
	n, err := w.WriteSamples(b.buf[b.r:b.w])
	if n < 0 {
		panic(errNegativeWrite)
	}
	b.r += n
	return int64(n), err
}
