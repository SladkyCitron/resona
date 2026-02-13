package generator_test

import (
	"fmt"

	"github.com/SladkyCitron/resona/freq"
	"github.com/SladkyCitron/resona/generator"
)

func ExampleConstant() {
	constant := generator.NewConstant(1)

	samples := make([]float32, 5)
	_, err := constant.ReadSamples(samples)
	if err != nil {
		panic(err)
	}

	fmt.Println(samples)
	fmt.Printf("%.2f == %.2f: %v\n", constant.Value, samples[0], constant.Value == samples[0])
	// Output:
	// [1 1 1 1 1]
	// 1.00 == 1.00: true
}

func ExampleNoise() {
	noise := generator.NewNoise()

	samples := make([]float32, 5)
	_, err := noise.ReadSamples(samples)
	if err != nil {
		panic(err)
	}

	fmt.Println(samples)
}

func ExampleOscillator() {
	sr := 48 * freq.KiloHertz
	osc := generator.NewOscillator(440*freq.Hertz, sr, generator.SineWaveform)

	samples := make([]float32, 5)
	_, err := osc.ReadSamples(samples)
	if err != nil {
		panic(err)
	}

	fmt.Println(samples)
}

func ExampleSilence() {
	var silence generator.Silence

	samples := make([]float32, 5)
	_, err := silence.ReadSamples(samples)
	if err != nil {
		panic(err)
	}

	fmt.Println(samples)
	// Output:
	// [0 0 0 0 0]
}
