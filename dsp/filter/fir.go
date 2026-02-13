package filter

import (
	"math"

	"github.com/SladkyCitron/resona/dsp/window"
	"github.com/SladkyCitron/resona/freq"
)

// FIR represents a FIR (Finite Impulse Response) filter.
type FIR struct {
	coeffs []float64
	buf    []float32
	pos    int
}

// NewFIR creates a new [FIR] filter.
func NewFIR(coeffs []float64) *FIR {
	return &FIR{
		coeffs: coeffs,
		buf:    make([]float32, len(coeffs)),
	}
}

// ProcessSingle processes a single input sample and returns the filtered output.
func (f *FIR) ProcessSingle(x float32) float32 {
	f.buf[f.pos] = x
	y := float32(0.0)
	j := f.pos
	for i := 0; i < len(f.coeffs); i++ {
		y += float32(f.coeffs[i]) * f.buf[j]
		j--
		if j < 0 {
			j = len(f.buf) - 1
		}
	}
	f.pos = (f.pos + 1) % len(f.buf)
	return y
}

// Reset resets internal state.
func (f *FIR) Reset() {
	f.buf = make([]float32, len(f.coeffs))
}

// DesignFIRLowpass designs a low-pass FIR filter using the windowed-sinc method.
// It returns the filter coefficients for use in [NewFIR].
func DesignFIRLowpass(cutoff, sampleRate freq.Frequency, taps int) []float64 {
	fc := cutoff.Hertz()
	fs := sampleRate.Hertz()

	coeffs := make([]float64, taps)
	m := taps - 1

	for n := range taps {
		if n == m/2 {
			coeffs[n] = 2 * fc / fs
		} else {
			x := math.Pi * (float64(n) - float64(m)/2.0)
			coeffs[n] = math.Sin(2*math.Pi*fc/fs*(float64(n)-float64(m)/2.0)) / x
		}
	}

	// Apply Hamming window
	window.MustApply(coeffs, window.Hamming)

	// Normalize
	sum := 0.0
	for _, c := range coeffs {
		sum += c
	}
	for i := range coeffs {
		coeffs[i] /= sum
	}

	return coeffs
}

// DesignFIRHighpass designs a high-pass FIR filter using spectral inversion.
// It returns the filter coefficients for use in [NewFIR].
func DesignFIRHighpass(cutoff, sampleRate freq.Frequency, taps int) []float64 {
	lpf := DesignFIRLowpass(cutoff, sampleRate, taps)
	hpf := make([]float64, taps)
	center := (taps - 1) / 2
	for n := range taps {
		if n == center {
			hpf[n] = 1.0 - lpf[n]
		} else {
			hpf[n] = -lpf[n]
		}
	}
	return hpf
}
