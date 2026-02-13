// Package codec provides utilities for working with audio codecs and encoding / decoding audio files.
//
// Similar to [image.Decode], decoding any particular
// audio format requires the prior registration of a decoder function.
// Registration is typically automatic as a side effect of initializing that
// format's package so that, to decode a MP3 file, it suffices to have
//
//	import _ "github.com/SladkyCitron/resona/codec/mp3"
//
// in a program's main package. The _ means to import a package purely for its
// initialization side effects.
package codec

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
)

// Decoder represents an abstract audio codec decoder.
type Decoder interface {
	aio.SampleReadSeeker
	afmt.Formatter
	afmt.SampleFormatter
	Len() int
}

// Bitrater is the interface with the Bitrate method.
type Bitrater interface {
	// Bitrate returns the bitrate of the audio stream in bytes per second.
	Bitrate() int
}

// ErrFormat indicates that decoding encountered an unknown format.
var ErrFormat = errors.New("codec: unknown format")

// A format holds an audio format's name, magic header and how to decode it.
type format struct {
	name, magic string
	decode      func(io.Reader) (Decoder, error)
}

// Formats is the list of registered formats.
var (
	formatsMu     sync.Mutex
	atomicFormats atomic.Value
)

// RegisterFormat registers an audio format for use by [Decode].
// Name is the name of the format, like "wav" or "mp3".
// Magic is the magic prefix that identifies the format's encoding. The magic
// string can contain "?" wildcards that each match any one byte.
// [Decode] is the function that decodes the encoded audio.
func RegisterFormat(name, magic string, decode func(io.Reader) (Decoder, error)) {
	formatsMu.Lock()
	formats, _ := atomicFormats.Load().([]format)
	atomicFormats.Store(append(formats, format{name, magic, decode}))
	formatsMu.Unlock()
}

// A reader is an io.Reader that can also peek ahead.
type reader interface {
	io.Reader
	Peek(int) ([]byte, error)
}

// asReader converts an io.Reader to a reader.
func asReader(r io.Reader) reader {
	if rr, ok := r.(reader); ok {
		return rr
	}
	return bufio.NewReader(r)
}

// match reports whether magic matches b. Magic may contain "?" wildcards.
func match(magic string, b []byte) bool {
	if len(magic) != len(b) {
		return false
	}
	for i, c := range b {
		if magic[i] != c && magic[i] != '?' {
			return false
		}
	}
	return true
}

// sniff determines the format of r's data.
func sniff(r reader) format {
	formats, _ := atomicFormats.Load().([]format)
	for _, f := range formats {
		b, err := r.Peek(len(f.magic))
		if err == nil && match(f.magic, b) {
			return f
		}
	}
	return format{}
}

// Decode decodes an audio stream that has been encoded in a registered format.
// The string returned is the format name used during format registration.
// Format registration is typically done by an init function in the codec- specific package.
func Decode(r io.Reader) (Decoder, string, error) {
	if sr, ok := r.(io.ReadSeeker); ok {
		br := bufio.NewReader(sr)
		f := sniff(br)
		if f.decode == nil {
			return nil, "", ErrFormat
		}
		if _, err := sr.Seek(0, io.SeekStart); err != nil {
			return nil, "", err
		}
		d, err := f.decode(sr)
		return d, f.name, err
	}

	rr := asReader(r)
	f := sniff(rr)
	if f.decode == nil {
		return nil, "", ErrFormat
	}
	d, err := f.decode(rr)
	return d, f.name, err
}
