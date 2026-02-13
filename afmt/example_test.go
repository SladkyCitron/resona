package afmt_test

import (
	"encoding/binary"
	"fmt"

	"github.com/SladkyCitron/resona/afmt"
)

func ExampleSampleFormat_String() {
	fmt.Println("8-bit unsigned PCM:", afmt.SampleFormat{BitDepth: 8, Encoding: afmt.SampleEncodingUint}.String())
	fmt.Println("16-bit signed big-endian PCM:", afmt.SampleFormat{BitDepth: 16, Encoding: afmt.SampleEncodingInt, Endian: binary.BigEndian}.String())
	fmt.Println("24-bit signed PCM:", afmt.SampleFormat{BitDepth: 24, Encoding: afmt.SampleEncodingInt}.String())
	fmt.Println("32-bit float little-endian PCM:", afmt.SampleFormat{BitDepth: 32, Encoding: afmt.SampleEncodingFloat, Endian: binary.LittleEndian}.String())
	// Output:
	// 8-bit unsigned PCM: uint8
	// 16-bit signed big-endian PCM: int16be
	// 24-bit signed PCM: int24
	// 32-bit float little-endian PCM: float32le
}
