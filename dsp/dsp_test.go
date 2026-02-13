package dsp_test

import (
	"testing"

	"github.com/SladkyCitron/resona/dsp"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestClamp(t *testing.T) {
	var want float32 = 1.0
	if got := dsp.Clamp(39.0); !testutil.EqualWithinTolerance(want, got, 1e-12) {
		t.Errorf("Clamp(39.0) = %v; want %v", got, want)
	}
}

func TestComplexFloatRoundtrip(t *testing.T) {
	want := []float32{1, 2, 3, 4, 5}

	c := dsp.ToComplexSlice(want)
	got := dsp.ToFloatSlice(c)

	if !testutil.EqualSliceWithinTolerance(want, got, 1e-12) {
		t.Errorf("Roundtrip failed: got %v; want %v", got, want)
	}
}
