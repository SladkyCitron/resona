package audio_test

import (
	"fmt"

	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/audio"
)

func ExampleInterleave() {
	stereo := [][]float32{
		{1.0, 1.0},
		{0.5, 0.5},
		{0.0, 0.0},
		{-1.0, -1.0},
	}

	interleaved := audio.Interleave(stereo)
	fmt.Printf("Interleaved: %v\n", interleaved)
	// Output:
	// Interleaved: [1 1 0.5 0.5 0 0 -1 -1]
}

func ExampleDeinterleave() {
	interleavedStereo := []float32{1.0, 1.0, 0.5, 0.5, 0.0, 0.0, -1.0, -1.0}
	numChannels := 2

	planar := audio.Deinterleave(interleavedStereo, numChannels)
	fmt.Printf("Deinterleaved: %v\n", planar)
	// Output:
	// Deinterleaved: [[1 0.5 0 -1] [1 0.5 0 -1]]
}

func ExampleBuffer() {
	var buf audio.Buffer // doesn't need any initialization
	buf.WriteSamples([]float32{1.0, 0.0})
	fmt.Printf("Samples: %v\n", buf.Float32s())
	// Output:
	// Samples: [1 0]
}

func ExampleBuffer_Cap() {
	buf := audio.NewBuffer(make([]float32, 0, 10))
	fmt.Println(buf.Cap())
	// Output:
	// 10
}

func ExampleBuffer_Len() {
	buf := audio.NewBuffer([]float32{1.0, 1.0, 0.5, 0.5, 0.0, 0.0, -1.0, -1.0})
	fmt.Println(buf.Len())
	// Output:
	// 8
}

func ExampleReader() {
	r := audio.NewReader([]float32{1.0, 0.0})
	samples, err := aio.ReadAll(r)
	if err != nil {
		panic(err)
	}
	fmt.Println(samples)
	// Output:
	// [1 0]
}

func ExampleDownmixer() {
	stereo := []float32{1.0, 1.0, 0.5, 0.5, 0.0, 0.0, -1.0, -1.0}
	downmix := audio.NewDownmixer(audio.NewReader(stereo), 2)
	mono, err := aio.ReadAll(downmix)
	if err != nil {
		panic(err)
	}
	fmt.Println(mono)
	// Output:
	// [1 0.5 0 -1]
}

func ExampleMixer() {
	mux := audio.NewMixer()
	mux.KeepAlive(false)
	r1 := audio.NewReader([]float32{0.1, 0.2, 0.3})
	r2 := audio.NewReader([]float32{0.1, 0.2, 0.3})
	mux.Add(r1, r2)

	samples, err := aio.ReadAll(mux)
	if err != nil {
		panic(err)
	}
	fmt.Println(samples)
	// Output:
	// [0.2 0.4 0.6]
}

func ExampleUpmixer() {
	// upmix mono => stereo
	mono := []float32{0.1, 0.2, 0.3}
	upmix := audio.NewUpmixer(audio.NewReader(mono), 2)
	stereo, err := aio.ReadAll(upmix)
	if err != nil {
		panic(err)
	}
	fmt.Println(stereo)
	// Output:
	// [0.1 0.1 0.2 0.2 0.3 0.3]
}
