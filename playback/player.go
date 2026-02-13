package playback

import (
	"sync"

	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/audio"
)

// Player represents an audio player.
type Player struct {
	mux *audio.Mixer
	src aio.SampleReader
}

// NewPlayer creates a new [Player].
func (ctx *Context) NewPlayer(src aio.SampleReader) *Player {
	return &Player{
		mux: ctx.mux,
		src: src,
	}
}

// Play starts the playback.
func (p *Player) Play() {
	p.mux.Add(p.src)
}

// PlayWithDone starts the playback and returns a channel that closes when the player has finished playing and drained.
func (p *Player) PlayWithDone() chan struct{} {
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)

	wrapped := aio.CallbackReader(p.src, func() {
		wg.Done()
	})

	go func() {
		wg.Wait()
		close(done)
	}()

	p.mux.Add(wrapped)

	return done
}

// PlayAndWait starts the playback and blocks until the player has finished playing and drained.
func (p *Player) PlayAndWait() {
	<-p.PlayWithDone()
}
