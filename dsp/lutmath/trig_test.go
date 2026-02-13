package lutmath_test

import (
	"math"
	"testing"

	"github.com/SladkyCitron/resona/dsp/lutmath"
	"github.com/SladkyCitron/resona/internal/testutil"
)

const testTolerance = 1e-3

func TestSin(t *testing.T) {
	want := float32(math.Sin(39))
	got := lutmath.Sin(39)
	if !testutil.EqualWithinTolerance(want, got, testTolerance) {
		t.Errorf("Sin(39) = %v, want %v", got, want)
	}
}

func TestCos(t *testing.T) {
	want := float32(math.Cos(39))
	got := lutmath.Cos(39)
	if !testutil.EqualWithinTolerance(want, got, testTolerance) {
		t.Errorf("Cos(39) = %v, want %v", got, want)
	}
}

func TestTan(t *testing.T) {
	want := float32(math.Tan(39))
	got := lutmath.Tan(39)
	if !testutil.EqualWithinTolerance(want, got, testTolerance) {
		t.Errorf("Tan(39) = %v, want %v", got, want)
	}
}
