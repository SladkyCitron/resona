package audio_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/SladkyCitron/resona/audio"
	"github.com/SladkyCitron/resona/freq"
)

func TestInterleave(t *testing.T) {
	tests := []struct {
		name  string
		input [][]float32
		want  []float32
	}{
		{
			name:  "Stereo",
			input: [][]float32{{0.1, 0.2}, {0.3, 0.4}, {0.5, 0.6}},
			want:  []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
		},
		{
			name:  "Mono",
			input: [][]float32{{0.1}, {0.2}, {0.3}},
			want:  []float32{0.1, 0.2, 0.3},
		},
		{
			name:  "EmptyInput",
			input: [][]float32{},
			want:  []float32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := audio.Interleave(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Interleave() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeinterleave(t *testing.T) {
	tests := []struct {
		name        string
		input       []float32
		numChannels int
		want        [][]float32
		shouldPanic bool
	}{
		{
			name:        "Stereo",
			input:       []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
			numChannels: 2,
			want: [][]float32{
				{0.1, 0.3, 0.5}, // Left
				{0.2, 0.4, 0.6}, // Right
			},
		},
		{
			name:        "Mono",
			input:       []float32{0.1, 0.2, 0.3},
			numChannels: 1,
			want: [][]float32{
				{0.1, 0.2, 0.3},
			},
		},
		{
			name:        "EmptyInput",
			input:       []float32{},
			numChannels: 2,
			want: [][]float32{
				{}, {},
			},
		},
		{
			name:        "InvalidChannelCount",
			input:       []float32{0.1, 0.2},
			numChannels: 0,
			shouldPanic: true,
		},
		{
			name:        "LengthNotDivisibleByNumChannels",
			input:       []float32{0.1, 0.2, 0.3},
			numChannels: 2,
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Deinterleave() did not panic as expected")
					}
				}()
			}

			got := audio.Deinterleave(tt.input, tt.numChannels)
			if !tt.shouldPanic && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Deinterleave() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockSeeker struct {
	position int64
}

func (s *mockSeeker) Seek(offset int64, whence int) (int64, error) {
	return s.position, nil
}

func TestPosition(t *testing.T) {
	s := &mockSeeker{position: 39} // Miku reference?! :3

	if pos := audio.Position(s); pos != 39 {
		t.Errorf("Position() = %d, want 39", pos)
	}
}

func TestPositionDur(t *testing.T) {
	s := &mockSeeker{position: 48000}
	sr := 48000 * freq.Hertz
	want := time.Second

	if dur := audio.PositionDur(s, sr); dur != want {
		t.Errorf("PositionDur() = %v, want %v", dur, want)
	}
}
