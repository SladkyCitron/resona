package window_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/SladkyCitron/resona/dsp/window"
	"github.com/SladkyCitron/resona/internal/testutil"
)

func TestWindows(t *testing.T) {
	lengths := []int{0, 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}

	funcs := map[string]window.WindowFunc{
		"Rectangular": window.Rectangular,
		"Welch":       window.Welch,
		"Hann":        window.Hann,
		"Hamming":     window.Hamming,
		"Blackman":    window.Blackman,
	}

	// Go maps are non-deterministic so using these function names so that it's in order
	funcNames := []string{
		"Rectangular",
		"Welch",
		"Hann",
		"Hamming",
		"Blackman",
	}

	for _, n := range lengths {
		t.Run(fmt.Sprintf("Length%d", n), func(t *testing.T) {
			f, err := os.Open(fmt.Sprintf("testdata/window%d.json", n))
			if err != nil {
				t.Fatalf("failed to open testdata file: %v", err)
			}
			defer f.Close()

			var data map[string][]float64
			if err := json.NewDecoder(f).Decode(&data); err != nil {
				t.Fatalf("failed to decode testdata file: %v", err)
			}

			for _, funcName := range funcNames {
				tt := struct {
					funcName string
					fn       window.WindowFunc
					n        int
					expected []float64
				}{
					funcName: funcName,
					fn:       funcs[funcName],
					n:        n,
					expected: data[funcName],
				}
				t.Run(fmt.Sprintf("%s%d", funcName, n), func(t *testing.T) {
					w := tt.fn(tt.n)
					if !testutil.EqualSliceWithinTolerance(tt.expected, w, 1e-9) {
						t.Errorf("%s(%d) does not match expected slice", tt.funcName, tt.n)
					}
				})
			}
		})
	}
}
