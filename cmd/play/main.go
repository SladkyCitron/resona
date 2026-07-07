package main

import (
	"fmt"
	"io"
	"os"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/audio"
	"github.com/SladkyCitron/resona/codec"
	_ "github.com/SladkyCitron/resona/codec/au"
	_ "github.com/SladkyCitron/resona/codec/avr"
	_ "github.com/SladkyCitron/resona/codec/flac"
	_ "github.com/SladkyCitron/resona/codec/mp3"
	_ "github.com/SladkyCitron/resona/codec/oggvorbis"
	_ "github.com/SladkyCitron/resona/codec/qoa"
	_ "github.com/SladkyCitron/resona/codec/svx"
	_ "github.com/SladkyCitron/resona/codec/wav"
	"github.com/SladkyCitron/resona/playback"
	_ "github.com/SladkyCitron/resona/playback/driver/oto"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <audio file>\n", os.Args[0])
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	dec, name, err := codec.Decode(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding file: %v\n", err)
		os.Exit(1)
	}

	format := dec.Format()
	fmt.Fprintf(os.Stderr, "Format: %s, %v, %d channels\n", name, format.SampleRate, format.NumChannels)

	if bitrater, ok := dec.(codec.Bitrater); ok {
		fmt.Fprintf(os.Stderr, "Bitrate: %d kbps\n", bitrater.Bitrate()/1000)
	}

	ctx, err := playback.NewContext(format, playback.WithDriver("oto"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating playback context: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	go func() {
		for {
			pos, _ := dec.Seek(0, io.SeekCurrent)
			fmt.Fprintf(os.Stderr, "\rPlaying... %v", afmt.NumFramesToDuration(format.SampleRate, int(pos)))
			if int(pos) >= dec.Len() {
				fmt.Println()
				return
			}
		}
	}()

	src := audio.NewSource(dec)
	player := ctx.NewPlayer(src)
	player.PlayAndWait()
}
