// Package audio implements functions and general-purpose utilities
// for the manipulation of sample slices and audio.
package audio

import (
	"io"
	"time"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/freq"
)

// Interleave takes a channel-separated audio sample slice, where each channel is a slice of samples,
// and combines them into a single interleaved slice.
//
// For example:
//
//	[][]float32{{L0, R0}, {L1, R1}, {L2, R2}, ...}
//
// becomes:
//
//	[]float32{L0, R0, L1, R1, L2, R2, ...}
func Interleave(samples [][]float32) []float32 {
	out := []float32{}
	for i := range samples {
		out = append(out, samples[i]...)
	}
	return out
}

// Deinterleave takes an interleaved audio slice and separates it into an individual channel-separated slice.
//
// For example:
//
//	[]float32{L0, R0, L1, R1, L2, R2, ...}
//
// becomes:
//
//	[][]float32{{L0, R0}, {L1, R1}, {L2, R2}, ...}
func Deinterleave(interleaved []float32, numChannels int) [][]float32 {
	if numChannels <= 0 {
		panic("audio: number of channels must be positive")
	}
	if len(interleaved)%numChannels != 0 {
		panic("audio: interleaved slice length is not divisible by number of channels")
	}

	totalFrames := len(interleaved) / numChannels
	out := make([][]float32, numChannels)
	for ch := range out {
		out[ch] = make([]float32, totalFrames)
	}

	for i := range totalFrames {
		for ch := range numChannels {
			out[ch][i] = interleaved[i*numChannels+ch]
		}
	}

	return out
}

// Position returns the current position of the given [io.Seeker] in frames.
func Position(s io.Seeker) int {
	pos, _ := s.Seek(0, io.SeekCurrent)
	return int(pos)
}

// PositionDur returns the current position of the given [io.Seeker] as a [time.Duration] given the sample rate.
func PositionDur(s io.Seeker, sampleRate freq.Frequency) time.Duration {
	return afmt.NumFramesToDuration(sampleRate, Position(s))
}
