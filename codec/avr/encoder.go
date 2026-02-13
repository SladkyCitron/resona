package avr

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/encoding/pcm"
)

// EncoderOption represents an option for configuring [Encoder].
type EncoderOption func(*Encoder)

// WithTitle sets the title.
func WithTitle(title [8]byte) EncoderOption {
	return func(e *Encoder) {
		e.title = title
	}
}

// WithExtraTitle sets the extra title.
func WithExtraTitle(extraTitle [20]byte) EncoderOption {
	return func(e *Encoder) {
		e.titleExtra = extraTitle
	}
}

// WithComment sets the comment.
func WithComment(comment [64]byte) EncoderOption {
	return func(e *Encoder) {
		e.comment = comment
	}
}

// WithLoop enables and configures the loop.
func WithLoop(start, end int) EncoderOption {
	return func(e *Encoder) {
		e.loop = -1
		e.loopStart = uint32(start)
		e.loopEnd = uint32(end)
	}
}

// Encoder represents the encoder for the AVR file format.
//
// The caller retains ownership of the writer; it will not be closed automatically.
type Encoder struct {
	w           io.WriteSeeker
	format      afmt.Format
	sampleFmt   afmt.SampleFormat
	enc         aio.SampleWriter
	title       [8]byte
	titleExtra  [20]byte
	comment     [64]byte
	loop        int16
	loopStart   uint32
	loopEnd     uint32
	dataWritten int
}

// NewEncoder creates a new [Encoder] for the AVR format.
func NewEncoder(w io.WriteSeeker, format afmt.Format, sampleFmt afmt.SampleFormat, opts ...EncoderOption) (*Encoder, error) {
	e := &Encoder{
		w:         w,
		format:    format,
		sampleFmt: sampleFmt,
	}

	// apply options
	for _, opt := range opts {
		opt(e)
	}

	// validate sample format
	if (sampleFmt.Encoding != afmt.SampleEncodingInt && sampleFmt.Encoding != afmt.SampleEncodingUint) ||
		(sampleFmt.Endian != nil && sampleFmt.Endian != binary.BigEndian) {
		return nil, fmt.Errorf("avr: invalid sample format: %s", sampleFmt.String())
	}
	if sampleFmt.Endian == nil {
		sampleFmt.Endian = binary.BigEndian
	}

	// validate loop info
	if e.loop == -1 && e.loopEnd <= e.loopStart {
		return nil, fmt.Errorf("avr: invalid loop range (%d-%d)", e.loopStart, e.loopEnd)
	}

	// write header
	if err := e.writeHeader(); err != nil {
		return nil, fmt.Errorf("avr: failed to write header: %w", err)
	}

	e.enc = pcm.NewEncoder(w, sampleFmt)

	return e, nil
}

func (e *Encoder) writeHeader() error {
	// write magic
	if _, err := e.w.Write([]byte(magic)); err != nil {
		return err
	}

	// write title
	if _, err := e.w.Write(e.title[:]); err != nil {
		return err
	}

	// write stereo
	var stereo int16
	if e.format.NumChannels == 2 {
		stereo = -1
	}
	if err := binary.Write(e.w, binary.BigEndian, stereo); err != nil {
		return err
	}

	// write bit depth
	if err := binary.Write(e.w, binary.BigEndian, int16(e.sampleFmt.BitDepth)); err != nil {
		return err
	}

	// write signed
	var signed int16
	if e.sampleFmt.Encoding == afmt.SampleEncodingInt {
		signed = -1
	}
	if err := binary.Write(e.w, binary.BigEndian, signed); err != nil {
		return err
	}

	// write loop
	if err := binary.Write(e.w, binary.BigEndian, e.loop); err != nil {
		return err
	}

	// write midiNote
	if _, err := e.w.Write([]byte{0, 0}); err != nil {
		return err
	}

	// write sample rate
	sr := uint32(e.format.SampleRate.Hertz()) & 0x00ffffff
	if err := binary.Write(e.w, binary.BigEndian, sr); err != nil {
		return err
	}

	// write length (placeholder)
	if err := binary.Write(e.w, binary.BigEndian, uint32(0xffffffff)); err != nil {
		return err
	}

	// write loop start
	if err := binary.Write(e.w, binary.BigEndian, e.loopStart); err != nil {
		return err
	}

	// write loop end
	if err := binary.Write(e.w, binary.BigEndian, e.loopEnd); err != nil {
		return err
	}

	// write key split, compression, and reserved
	if _, err := e.w.Write([]byte{0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}

	// write extra title
	if _, err := e.w.Write(e.titleExtra[:]); err != nil {
		return err
	}

	// write comment
	if _, err := e.w.Write(e.comment[:]); err != nil {
		return err
	}

	return nil
}

// WriteSamples encodes and writes samples.
func (e *Encoder) WriteSamples(p []float32) (int, error) {
	n, err := e.enc.WriteSamples(p)
	e.dataWritten += n
	return n, err
}

// Close finalizes the encoding process and writes the length value.
//
// It will NOT close the underlying writer, even if it implements [io.Closer].
// Closing the underlying writer is the owner's responsibility.
func (e *Encoder) Close() error {
	if _, err := e.w.Seek(26, io.SeekStart); err != nil {
		return fmt.Errorf("avr: failed to seek to length: %w", err)
	}

	if err := binary.Write(e.w, binary.BigEndian, uint32(e.dataWritten)); err != nil {
		return fmt.Errorf("avr: failed to write data size: %w", err)
	}

	if _, err := e.w.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("avr: failed to seek to end: %w", err)
	}

	return nil
}
