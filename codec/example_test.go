package codec_test

import (
	"fmt"
	"os"

	"github.com/SladkyCitron/resona/codec"
	_ "github.com/SladkyCitron/resona/codec/flac" // Enable FLAC decoder
)

func Example() {
	f, err := os.Open("file.flac")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	dec, name, err := codec.Decode(f)
	if err != nil {
		panic(err)
	}

	fmt.Println(name) // Should print "flac"
	_ = dec           // Do something with the audio

	// Print out bitrate if the decoder supports it
	if bitrater, ok := dec.(codec.Bitrater); ok {
		fmt.Printf("Bitrate: %d kbps\n", bitrater.Bitrate()/1000)
	}
}
