package playback

import (
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/playback/driver"
)

// Player represents an audio player.
type Player struct {
	driver.Player
}

// NewPlayer creates a new [Player].
func (ctx *Context) NewPlayer(src aio.SampleReader) *Player {
	return &Player{Player: ctx.drv.NewPlayer(src)}
}

// PlayAndWait starts the playback and blocks until the player has finished playing and drained.
func (p *Player) PlayAndWait() {
	<-p.PlayWithDone()
}
