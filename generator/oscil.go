package generator

import (
	"math"

	"github.com/SladkyCitron/resona/dsp/lutmath"
	"github.com/SladkyCitron/resona/freq"
)

const tau = 2 * math.Pi

// OscilWaveform is a function that maps a waveform value for a given input x in the range [0, 1).
type OscilWaveform func(x float32) float32

// SineWaveform generates a sine wave.
func SineWaveform(x float32) float32 {
	return float32(math.Sin(tau * float64(x)))
}

// LUTSineWaveform generates a sine wave using a LUT (see lutmath package).
func LUTSineWaveform(x float32) float32 {
	return lutmath.Sin(tau * x)
}

// TriangleWaveform generates a triangle wave.
func TriangleWaveform(x float32) float32 {
	return float32(1 - 4*math.Abs(math.Round(float64(x)-0.25)-float64(x)+0.25))
}

// SawtoothWaveform generates a sawtooth wave.
func SawtoothWaveform(x float32) float32 {
	return float32(2 * (float64(x) - math.Floor(float64(x)+0.5)))
}

// SquareWaveform generates a square wave.
func SquareWaveform(x float32) float32 {
	return float32(math.Copysign(1, math.Sin(tau*float64(x))))
}

// LUTSquareWaveform generates a square wave using a LUT (see lutmath package).
func LUTSquareWaveform(x float32) float32 {
	return float32(math.Copysign(1, float64(lutmath.Sin(tau*x))))
}

// TODO: modulation via Parameters???

// Oscillator is a simple oscillator that generates a waveform at a specified frequency and sample rate.
type Oscillator struct {
	Frequency  freq.Frequency
	sampleRate freq.Frequency
	waveform   OscilWaveform
	t          float32
}

// NewOscillator creates a new [Oscillator].
func NewOscillator(f freq.Frequency, sampleRate freq.Frequency, waveform OscilWaveform) *Oscillator {
	return &Oscillator{
		Frequency:  f,
		sampleRate: sampleRate,
		waveform:   waveform,
	}
}

func (o *Oscillator) ReadSamples(p []float32) (int, error) {
	for i := range p {
		p[i] = o.waveform(o.t * float32(o.Frequency.Hertz()/2/o.sampleRate.Hertz()))
		o.t++
	}
	return len(p), nil
}
