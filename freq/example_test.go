package freq_test

import (
	"fmt"
	"time"

	"github.com/SladkyCitron/resona/freq"
)

func ExampleParse() {
	kilohertz, err := freq.Parse("1kHz")
	if err != nil {
		panic(err)
	}

	fmt.Println(kilohertz)
	// Output:
	// 1kHz
}

func ExampleFrequency_Abs() {
	positive := 2 * freq.KiloHertz
	negative := -3 * freq.KiloHertz

	absPositive := positive.Abs()
	absNegative := negative.Abs()

	fmt.Printf("Absolute value of positive frequency: %v\n", absPositive)
	fmt.Printf("Absolute value of negative frequency: %v\n", absNegative)
	// Output:
	// Absolute value of positive frequency: 2kHz
	// Absolute value of negative frequency: 3kHz
}

func ExampleFrequency_Hertz() {
	fmt.Printf("There are %.0f Hz in one kilohertz\n", freq.KiloHertz.Hertz())
	fmt.Printf("There are %.0f Hz in one megahertz\n", freq.MegaHertz.Hertz())
	// Output:
	// There are 1000 Hz in one kilohertz
	// There are 1000000 Hz in one megahertz
}

func ExampleFrequency_Period() {
	f := 2 * freq.Hertz
	p := f.Period()
	fmt.Println(p)
	// Output:
	// 500ms
}

func ExampleFromPeriod() {
	p := 200 * time.Millisecond
	f := freq.FromPeriod(p)
	fmt.Println(f)
	// Output:
	// 5Hz
}
