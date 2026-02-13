package avr

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/codec"
	"github.com/SladkyCitron/resona/encoding/pcm"
	"github.com/SladkyCitron/resona/freq"
)

// Decoder represents the decoder of the AVR file format.
// It implements codec.Decoder.
type Decoder struct {
	r        io.Reader
	dataRead int

	dec aio.SampleReader

	title    [8]byte
	stereo   int16
	bitDepth int16
	signed   int16
	loop     int16
	//midiNote int16
	sampleRate uint32
	length     uint32 // Length in frames (groups of samples)
	loopStart  uint32
	loopEnd    uint32
	//keySplit int16
	//compression int16
	//reserved int16
	titleExtra [20]byte
	comment    [64]byte
}

// NewDecoder creates a new [Decoder] and decodes the headers.
func NewDecoder(r io.Reader) (codec.Decoder, error) {
	d := &Decoder{r: r}

	// Read magic
	var buf [len(magic)]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, fmt.Errorf("avr: failed to read magic: %w", err)
	}
	if string(buf[:]) != magic {
		return nil, errors.New("avr: invalid header")
	}

	// Read title
	if _, err := io.ReadFull(r, d.title[:]); err != nil {
		return nil, fmt.Errorf("avr: failed to read title: %w", err)
	}

	// Read stereo
	if err := binary.Read(r, binary.BigEndian, &d.stereo); err != nil {
		return nil, fmt.Errorf("avr: failed to read stereo value: %w", err)
	}

	// Read bits
	if err := binary.Read(r, binary.BigEndian, &d.bitDepth); err != nil {
		return nil, fmt.Errorf("avr: failed to read bit depth: %w", err)
	}

	// Read signed
	if err := binary.Read(r, binary.BigEndian, &d.signed); err != nil {
		return nil, fmt.Errorf("avr: failed to read signed value: %w", err)
	}

	// Read loop
	if err := binary.Read(r, binary.BigEndian, &d.loop); err != nil {
		return nil, fmt.Errorf("avr: failed to read loop value: %w", err)
	}

	// Skip midiNote
	if _, err := io.CopyN(io.Discard, r, 2); err != nil { // int16 = 2 bytes
		return nil, err
	}

	// Read sample rate
	if err := binary.Read(r, binary.BigEndian, &d.sampleRate); err != nil {
		return nil, fmt.Errorf("avr: failed to read sample rate: %w", err)
	}
	d.sampleRate &= 0x00ffffff // mask off upper byte

	// Read length
	if err := binary.Read(r, binary.BigEndian, &d.length); err != nil {
		return nil, fmt.Errorf("avr: failed to read length: %w", err)
	}

	// Read loop start
	if err := binary.Read(r, binary.BigEndian, &d.loopStart); err != nil {
		return nil, fmt.Errorf("avr: failed to read loop start")
	}

	// Read loop end
	if err := binary.Read(r, binary.BigEndian, &d.loopEnd); err != nil {
		return nil, fmt.Errorf("avr: failed to read loop end")
	}

	// Skip key split, compression, and reserved
	if _, err := io.CopyN(io.Discard, r, 6); err != nil { // int16 = 2 bytes
		return nil, err
	}

	// Read extra title
	if _, err := io.ReadFull(r, d.titleExtra[:]); err != nil {
		return nil, fmt.Errorf("avr: failed to read extra title: %w", err)
	}

	// Read comment
	if _, err := io.ReadFull(r, d.comment[:]); err != nil {
		return nil, fmt.Errorf("avr: failed to read comment: %w", err)
	}

	d.dec = pcm.NewDecoder(r, d.SampleFormat())

	return d, nil
}

// AVR-specific values

// Title returns the title, padded with 0s.
func (d *Decoder) Title() [8]byte {
	return d.title
}

// TitleString returns the title as a string with trimmed 0s.
func (d *Decoder) TitleString() string {
	return string(bytes.Trim(d.title[:], string([]byte{0})))
}

// Loop returns whether looping is enabled.
func (d *Decoder) Loop() bool {
	switch d.loop {
	case -1:
		return true
	case 0:
		return false
	default:
		return false
	}
}

// LoopStart returns the loop start point in frames.
func (d *Decoder) LoopStart() int {
	return int(d.loopStart)
}

// LoopEnd returns the loop end point in frames.
func (d *Decoder) LoopEnd() int {
	return int(d.loopEnd)
}

// ExtraTitle returns the extra title, padded with 0s.
func (d *Decoder) ExtraTitle() [20]byte {
	return d.titleExtra
}

// ExtraTitleString returns the extra title as a string with trimmed 0s.
func (d *Decoder) ExtraTitleString() string {
	return string(bytes.Trim(d.titleExtra[:], string([]byte{0})))
}

// Comment returns the comment, padded with 0s.
func (d *Decoder) Comment() [64]byte {
	return d.comment
}

// CommentString returns the comment as a string with trimmed 0s.
func (d *Decoder) CommentString() string {
	return string(bytes.Trim(d.comment[:], string([]byte{0})))
}

// end AVR-specific values

// Format returns the audio stream format.
func (d *Decoder) Format() afmt.Format {
	f := afmt.Format{
		SampleRate:  freq.Frequency(d.sampleRate) * freq.Hertz,
		NumChannels: 1,
	}
	if d.stereo == -1 {
		f.NumChannels = 2
	}
	return f
}

// SampleFormat returns the sample format.
func (d *Decoder) SampleFormat() afmt.SampleFormat {
	f := afmt.SampleFormat{
		BitDepth: int(d.bitDepth),
		Encoding: afmt.SampleEncodingUint,
		Endian:   binary.BigEndian,
	}
	if d.signed == -1 {
		f.Encoding = afmt.SampleEncodingInt
	}
	return f
}

// Len returns the total number of frames.
func (d *Decoder) Len() int {
	return int(d.length)
}

// ReadSamples reads float32 samples into p.
// It returns the number of samples read and/or an error.
func (d *Decoder) ReadSamples(p []float32) (int, error) {
	n, err := d.dec.ReadSamples(p)
	d.dataRead += n * (d.SampleFormat().BitDepth / 8)
	return n, err
}

// Seek seeks to the specified frame.
// It returns the new offset relative to the start and/or an error.
// It will return an error if the source is not an [io.Seeker].
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	s, ok := d.r.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("avr: resource does not support seeking")
	}

	numChans := d.Format().NumChannels

	// Special case
	if offset == 0 && whence == io.SeekCurrent {
		return int64(d.dataRead) / int64(d.SampleFormat().BytesPerFrame(numChans)), nil
	}

	frameSize := d.SampleFormat().BytesPerFrame(numChans)
	totalFrames := int64(d.length)

	var target int64
	switch whence {
	case io.SeekStart:
		target = offset
	case io.SeekCurrent:
		target = int64(d.dataRead)/int64(frameSize) + offset
	case io.SeekEnd:
		target = int64(totalFrames) + offset
	default:
		return 0, fmt.Errorf("avr: invalid seek whence")
	}

	if target < 0 || target > totalFrames {
		return 0, fmt.Errorf("avr: seek out of bounds")
	}

	byteOffset := target * int64(frameSize)

	_, err := s.Seek(byteOffset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("avr: failed to seek: %w", err)
	}

	d.dataRead = int(byteOffset)
	return target, nil
}

func init() {
	codec.RegisterFormat("avr", magic, NewDecoder)
}
