package g711_test

import (
	"testing"

	"github.com/SladkyCitron/resona/encoding/g711"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestUlawRoundTrip(t *testing.T) {
	samples := []float32{0, 0.5, -0.5, 1, -1}

	encoded := g711.EncodeUlaw(samples)
	decoded := g711.DecodeUlaw(encoded)

	if len(decoded) != len(samples) {
		t.Fatalf("sample count mismatch: got %d, want %d", len(decoded), len(samples))
	}

	if !testutil.EqualSliceWithinTolerance(decoded, samples, 1) {
		t.Errorf("Decoded samples do not match original samples: got %v, want %v", decoded, samples)
	}
}
