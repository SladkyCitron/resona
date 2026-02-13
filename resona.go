// Package resona is the audio and DSP toolkit for Go.
//
// # Interleaved Audio Format
//
// All sample slices in Resona represent interleaved, multi-channel audio data.
// In an interleaved layout, samples for each channel are stored in sequence for each frame.
// For example, in stereo (2-channel) audio, the slice:
//
//	[]float32{L0, R0, L1, R1, L2, R2, ...}
//
// contains successive frames, where each frame consists of one sample per channel (left and right).
// This layout is common in many audio APIs and is efficient for streaming and hardware buffers.
// However, this may be less convenient for certain processing tasks.
// If channel-separated (planar) access is required, callers may convert interleaved slices using the audio package.
//
// The number of channels is not specified by the aio package interfaces themselves;
// it is an implicit contract between the caller and the implementation.
// Implementations should clearly document the expected or provided number of channels
// and sample rate (e.g. using the afmt package).
//
// # Sample Format
//
// All samples are represented as 32-bit floating-point numbers (float32).
// The value range is typically normalized between -1.0 and +1.0, where:
//
//   - 0.0 represents silence
//   - -1.0 to +1.0 represents full-scale audio signal
//   - Values outside this range may be clipped or distorted depending on the backend
//
// # Seeking
//
// All seekable streams implement the [io.Seeker] interface.
// Seek offset is measured in frames.
package resona

import (
	"fmt"
	"runtime/debug"
)

const root string = "github.com/SladkyCitron/resona"

// Version returns the version of Resona and its checksum. The returned
// values are only valid in binaries built with module support.
//
// If a replace directive exists in the Resona go.mod, the replace will
// be reported in the version in the following format:
//
//	"version=>[replace-path] [replace-version]"
//
// and the replace sum will be returned in place of the original sum.
func Version() (version, sum string) {
	b, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}
	for _, m := range b.Deps {
		if m.Path == root {
			if m.Replace != nil {
				switch {
				case m.Replace.Version != "" && m.Replace.Path != "":
					return fmt.Sprintf("%s=>%s %s", m.Version, m.Replace.Path, m.Replace.Version), m.Replace.Sum
				case m.Replace.Version != "":
					return fmt.Sprintf("%s=>%s", m.Version, m.Replace.Version), m.Replace.Sum
				case m.Replace.Path != "":
					return fmt.Sprintf("%s=>%s", m.Version, m.Replace.Path), m.Replace.Sum
				default:
					return m.Version + "*", m.Sum + "*"
				}
			}
			return m.Version, m.Sum
		}
	}
	return "", ""
}
