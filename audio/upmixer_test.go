package audio_test

import (
	"testing"

	"github.com/SladkyCitron/resona/audio"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestUpmixer(t *testing.T) {
	samples := []float32{0.1, 0.2, 0.3}
	want := []float32{0.1, 0.1, 0.2, 0.2, 0.3, 0.3}

	upmix := audio.NewUpmixer(audio.NewReader(samples), 2)

	got := make([]float32, len(samples)*2)
	_, err := upmix.ReadSamples(got)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.EqualSliceWithinTolerance(want, got, 1e-12) {
		t.Errorf("upmixed samples do not match: got %v, want %v", got, want)
	}
}
