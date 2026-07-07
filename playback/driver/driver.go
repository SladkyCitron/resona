// Package driver provides the interface for playback drivers.
package driver

import (
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
)

// Driver is the interface that playback drivers must implement.
type Driver interface {
	// Init initializes the driver with the given format and buffer size.
	Init(format afmt.Format, bufferSize int) error

	// NewPlayer creates a new [Player].
	NewPlayer(src aio.SampleReader) Player

	io.Closer
}

type Player interface {
	// Play starts playback.
	Play()

	// PlayWithDone starts the playback and returns a channel that closes when the player has finished playing and drained.
	PlayWithDone() chan struct{}
}
