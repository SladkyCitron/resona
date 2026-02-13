package fourier_test

import (
	"testing"

	"github.com/SladkyCitron/resona/dsp/fourier"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestRFFTRoundtrip(t *testing.T) {
	want := []float32{-1, -1, 1, 1, -1, -1, 1, 1}

	fft := fourier.RFFT(want)
	got := fourier.IRFFT(fft)

	if !testutil.EqualSliceWithinTolerance(got, want, 1e-3) {
		t.Errorf("FFT roundtrip failed: got %v, want %v", got, want)
	}
}

func TestRFFTInvalidLength(t *testing.T) {
	x := []float32{0, 1, 39} // length is not a power of two

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("RFFT did not panic on invalid length input")
		}
	}()

	_ = fourier.RFFT(x)
}

func TestIRFFTInvalidLength(t *testing.T) {
	x := []complex64{0, 1, 39, 42} // N is not a power of two (4/2+1 = 3)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("IRFFT did not panic on invalid length input")
		}
	}()

	_ = fourier.IRFFT(x)
}
