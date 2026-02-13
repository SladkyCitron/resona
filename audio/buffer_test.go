package audio_test

import (
	"testing"

	"github.com/SladkyCitron/resona/audio"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestBufferReadWriteRoundtrip(t *testing.T) {
	var buf audio.Buffer
	want := []float32{1, 2, 3, 4, 5}

	n, err := buf.WriteSamples(want)
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	if n != len(want) {
		t.Fatalf("write length = %d; want %d", n, len(want))
	}

	got := make([]float32, len(want))
	n, err = buf.ReadSamples(got)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if n != len(want) {
		t.Fatalf("read length = %d; want %d", n, len(want))
	}

	if !testutil.EqualSliceWithinTolerance(got, want, 1e-12) {
		t.Fatalf("buffer roundtrip failed: want %v, got %v", want, got)
	}
}
