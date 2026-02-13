package filter_test

import (
	"testing"

	"github.com/SladkyCitron/resona/dsp/filter"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestFIRImpulseResponse(t *testing.T) {
	coeffs := []float64{0.1, 0.2, 0.3, 0.5, 0.6}
	coeffsf32 := []float32{0.1, 0.2, 0.3, 0.5, 0.6}
	fir := filter.NewFIR(coeffs)

	input := []float32{1, 0, 0, 0, 0} // impulse
	got := make([]float32, len(input))
	for i := range input {
		got[i] = fir.ProcessSingle(input[i])
	}

	if !testutil.EqualSliceWithinTolerance(coeffsf32, got, 1e-12) {
		t.Errorf("output does not equal to coefficients: want %v, got %v", coeffsf32, got)
	}
}
