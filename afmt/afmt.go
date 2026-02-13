// Package afmt provides utilities for working with audio formats.
package afmt

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/SladkyCitron/resona/freq"
)

// NumFramesToDuration returns the duration of n frames based on the sample rate.
func NumFramesToDuration(sampleRate freq.Frequency, n int) time.Duration {
	return time.Second * time.Duration(n) / time.Duration(sampleRate.Hertz())
}

// DurationToNumFrames returns the number of frames that last for d duration based on the sample rate.
func DurationToNumFrames(sampleRate freq.Frequency, d time.Duration) int {
	return int(d * time.Duration(sampleRate.Hertz()) / time.Second)
}

// Format represents an abstract audio stream format.
type Format struct {
	// SampleRate is the sample rate.
	SampleRate freq.Frequency

	// NumChannels specifies the number of audio channels.
	// For example, 1 for mono, 2 for stereo. Samples are always interleaved (see package aio).
	NumChannels int
}

// Formatter is an interface for types that can report their audio format.
type Formatter interface {
	// Format returns the audio stream format.
	Format() Format
}

// SampleEncoding represents the encoding type of a single audio sample.
type SampleEncoding uint8

const (
	// SampleEncodingUnknown indicates an unknown or unspecified sample encoding.
	SampleEncodingUnknown SampleEncoding = iota

	// SampleEncodingInt represents signed integer-encoded samples (e.g., int16, int32).
	SampleEncodingInt

	// SampleEncodingUint represents unsigned integer-encoded samples (e.g., uint8).
	SampleEncodingUint

	// SampleEncodingFloat represents IEEE 754 float-encoded samples (e.g., float32, float64).
	SampleEncodingFloat
)

// IsSigned returns true if the sample encoding is signed (only SampleEncodingInt).
func (e SampleEncoding) IsSigned() bool {
	return e == SampleEncodingInt
}

// IsFloat returns true if the sample encoding is a floating-point format.
func (e SampleEncoding) IsFloat() bool {
	return e == SampleEncodingFloat
}

// IsInt returns true if the sample encoding is an integer format (signed or unsigned).
func (e SampleEncoding) IsInt() bool {
	return e == SampleEncodingInt || e == SampleEncodingUint
}

// SampleFormat describes the binary representation of individual audio samples.
type SampleFormat struct {
	BitDepth int              // BitDepth is the number of bits used to store each sample (e.g., 16, 24, 32).
	Encoding SampleEncoding   // Encoding specifies how the sample is stored (e.g., integer, float).
	Endian   binary.ByteOrder // Endian specifies the byte order (big or little endian). May be nil if not applicable.
}

// BytesPerSample returns the number of bytes used to store one mono sample based on its format.
// It rounds up to the nearest whole byte.
func (f SampleFormat) BytesPerSample() int {
	if f.BitDepth <= 0 {
		return 0
	}
	switch f.Encoding {
	case SampleEncodingInt, SampleEncodingUint, SampleEncodingFloat:
		return (f.BitDepth + 7) / 8
	default:
		return 0
	}
}

// BytesPerFrame returns the number of bytes used to store one multi-channel frame based in its format.
func (f SampleFormat) BytesPerFrame(numChannels int) int {
	return f.BytesPerSample() * numChannels
}

func (f SampleFormat) String() string {
	var s string

	switch f.Encoding {
	case SampleEncodingInt:
		s += "int"
	case SampleEncodingUint:
		s += "uint"
	case SampleEncodingFloat:
		s += "float"
	default:
		s += "unknown"
	}

	if f.BitDepth > 0 {
		s += fmt.Sprint(f.BitDepth)
	}

	switch f.Endian {
	case binary.LittleEndian:
		s += "le"
	case binary.BigEndian:
		s += "be"
	default:
		// No endian specified.
	}

	return s
}

// SampleFormatter is an interface for types that can report their sample format.
type SampleFormatter interface {
	// SampleFormat returns the sample format.
	SampleFormat() SampleFormat
}
