package audio_test

import (
	"testing"

	"github.com/SladkyCitron/resona/audio"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestMixer(t *testing.T) {
	src1 := audio.NewReader([]float32{0.1, 0.2, 0.3, 0.0})
	src2 := audio.NewReader([]float32{0.2, 0.3, 0.4, 0.0})
	want := []float32{0.3, 0.5, 0.7, 0.0}

	mixer := audio.NewMixer(src1)
	mixer.Add(src2)
	got := make([]float32, len(want))
	_, err := mixer.ReadSamples(got)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.EqualSliceWithinTolerance(got, want, 1e-3) {
		t.Errorf("mixer: got %v, want %v", got, want)
	}
}
