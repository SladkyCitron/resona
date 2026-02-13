package playback_test

import (
	"os"

	"github.com/SladkyCitron/resona/codec"
	_ "github.com/SladkyCitron/resona/codec/wav" // Enable WAV decoder
	"github.com/SladkyCitron/resona/playback"
	_ "github.com/SladkyCitron/resona/playback/driver/oto" // Enable Oto driver
)

func Example() {
	// Open the audio file
	f, err := os.Open("file.wav")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Decode audio file into an aio.SampleReader
	decoder, _, err := codec.Decode(f)
	if err != nil {
		panic(err)
	}

	// Create audio playback context with Oto as the driver
	ctx, err := playback.NewContext(decoder.Format(), playback.WithDriver("oto"))
	if err != nil {
		panic(err)
	}
	defer ctx.Close()

	// Create player and start playback
	player := ctx.NewPlayer(decoder)
	player.PlayAndWait() // Play and block until done
}
