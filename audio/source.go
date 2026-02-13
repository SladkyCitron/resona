package audio

import (
	"io"
	"math"
	"time"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/effect"
)

// Source represents a controllable audio stream.
//
// It wraps an aio.SampleReader and provides high-level playback controls
// and utilities such as pausing, muting, seeking, and volume control.
// Internally, Source uses an effects.Chain with Mute and Gain, so that
// the user does not need to compose them manually.
type Source struct {
	r        aio.SampleReader
	pausable *aio.PausableReader
	mute     *effect.Mute
	gain     *effect.Gain
}

// NewSource creates a new [Source] from the given reader.
// It automatically wraps the reader with mute and gain effects, and makes it pausable.
func NewSource(r aio.SampleReader) *Source {
	mute := &effect.Mute{}
	gain := &effect.Gain{}

	return &Source{
		r:        r,
		pausable: aio.NewPausableReader(effect.Reader(r, effect.Chain{mute, gain})),
		mute:     mute,
		gain:     gain,
	}
}

// Format returns the audio stream format.
func (s *Source) Format() afmt.Format {
	return s.r.(afmt.Formatter).Format()
}

// SampleFormat returns the sample format.
func (s *Source) SampleFormat() afmt.SampleFormat {
	return s.r.(afmt.SampleFormatter).SampleFormat()
}

// Seek seeks to the specified frame.
// It returns the new offset relative to the start and/or an error.
func (s *Source) Seek(offset int64, whence int) (int64, error) {
	return s.r.(io.Seeker).Seek(offset, whence)
}

// Position returns the current position in frames.
func (s *Source) Position() int {
	return Position(s.r.(io.Seeker))
}

// PositionDur returns the current position as a [time.Duration] given the audio format and sample rate.
func (s *Source) PositionDur() time.Duration {
	return PositionDur(s.r.(io.Seeker), s.Format().SampleRate)
}

// Pause pauses the audio stream. While paused, it outputs silence.
func (s *Source) Pause() {
	s.pausable.Pause()
}

// Resume resumes the audio stream.
func (s *Source) Resume() {
	s.pausable.Resume()
}

// IsPaused reports whether the audio stream is currently paused.
func (s *Source) IsPaused() bool {
	return s.pausable.IsPaused()
}

// Mute mutes the audio stream.
func (s *Source) Mute() {
	s.mute.Mute = true
}

// Unmute unmutes the audio stream.
func (s *Source) Unmute() {
	s.mute.Mute = false
}

// ToggleMute toggles the mute state.
func (s *Source) ToggleMute() {
	s.mute.Mute = !s.mute.Mute
}

// IsMuted reports whether the audio stream is muted.
func (s *Source) IsMuted() bool {
	return s.mute.Mute
}

// Volume returns the current volume as a linear gain value.
func (s *Source) Volume() float64 {
	return s.gain.Gain
}

// SetVolume sets the volume as a linear gain value.
// A value of 1.0 is unity gain, 0.5 is half volume, values >1.0 amplify.
func (s *Source) SetVolume(gain float64) {
	s.gain.Gain = gain
}

// SetVolumeDB sets the volume using a decibel (dB) value.
// 0.0 dB is unity gain, -6.0 dB is roughly half perceived loudness.
func (s *Source) SetVolumeDB(dB float64) {
	s.gain.Gain = math.Pow(10, dB/20)
}

//TODO: maybe pan

// ReadSamples reads audio samples into p from the underlying stream.
// It passes through the mute/gain effects and respects pause/resume state.
func (s *Source) ReadSamples(p []float32) (n int, err error) {
	return s.pausable.ReadSamples(p)
}
