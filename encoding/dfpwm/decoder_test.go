package dfpwm_test

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/encoding/dfpwm"
	"github.com/SladkyCitron/resona/encoding/pcm"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestDecode(t *testing.T) {
	wantBytes, err := os.ReadFile("testdata/sine.pcm")
	if err != nil {
		t.Fatal(err)
	}

	want, err := pcm.Decode(wantBytes, afmt.SampleFormat{BitDepth: 32, Encoding: afmt.SampleEncodingFloat, Endian: binary.LittleEndian})
	if err != nil {
		t.Fatal(err)
	}

	encBytes, err := os.ReadFile("testdata/sine.dfpwm")
	if err != nil {
		t.Fatal(err)
	}

	got, err := dfpwm.Decode(encBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}

	if !testutil.EqualSliceWithinTolerance(got, want, 0.1) {
		t.Fatalf("decoded samples do not match expected, got %v..., want %v...", got[:50], want[:50])
	}
}

func TestDecodeEmpty(t *testing.T) {
	got, err := dfpwm.Decode(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected empty output, got length %d", len(got))
	}
}

func TestDecodedLen(t *testing.T) {
	if x := dfpwm.DecodedLen(1); x != 8 {
		t.Fatalf("DecodedLen(1) = %d; want 8", x)
	}
}
