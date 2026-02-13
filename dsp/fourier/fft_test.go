package fourier_test

import (
	"testing"

	"github.com/SladkyCitron/resona/dsp/fourier"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestFFTRoundtrip(t *testing.T) {
	want := []complex64{-1, -1, 1, 1, -1, -1, 1, 1}

	fft := fourier.FFT(want)
	got := fourier.IFFT(fft)

	if !testutil.EqualSliceWithinTolerance(got, want, 1e-3) {
		t.Errorf("FFT roundtrip failed: got %v, want %v", got, want)
	}
}

func TestFFTInvalidLength(t *testing.T) {
	x := []complex64{0, 1, 39} // length is not a power of two

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("FFT did not panic on invalid length input")
		}
	}()

	fourier.FFTInPlace(x)
}

func TestIFFTInvalidLength(t *testing.T) {
	x := []complex64{0, 1, 39} // length is not a power of two

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("IFFT did not panic on invalid length input")
		}
	}()

	fourier.IFFTInPlace(x)
}

func TestConvolveImpulse(t *testing.T) {
	x := []complex64{0, 1, 0, 0, 0, 0, 0, 0}
	y := []complex64{1, 2, 3, 4, 5, 6, 7, 8}
	want := []complex64{8, 1, 2, 3, 4, 5, 6, 7}

	got := fourier.Convolve(x, y)
	if !testutil.EqualSliceWithinTolerance(got, want, 1e-3) {
		t.Errorf("Convolve with impulse failed: got %v, want %v", got, want)
	}

}

func TestConvolveInvalidLength(t *testing.T) {
	x := []complex64{0, 1, 39}      // length is not a power of two
	y := []complex64{1, 2, 3, 4, 5} // x and y lengths do not match

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Convolve did not panic on invalid length input")
		}
	}()

	_ = fourier.Convolve(x, y)
}
