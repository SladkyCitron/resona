package playback

import (
	"fmt"

	"github.com/SladkyCitron/resona/abufio"
	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/audio"
	"github.com/SladkyCitron/resona/playback/driver"
)

// ContextOption represents an option for configuring [Context].
type ContextOption func(*Context)

// WithDriver sets the playback driver.
func WithDriver(name string) ContextOption {
	return func(ctx *Context) {
		ctx.driverName = name
	}
}

// WithBufferSize sets the buffer size.
// Bigger buffer size means lower CPU usage and more reliable playback.
// Lower buffer size means better responsiveness and less delay.
func WithBufferSize(size int) ContextOption {
	return func(ctx *Context) {
		ctx.bufferSize = size
	}
}

// WithBufferSizeBytes sets the buffer size in bytes.
// Bigger buffer size means lower CPU usage and more reliable playback.
// Lower buffer size means better responsiveness and less delay.
func WithBufferSizeBytes(size int) ContextOption {
	return func(ctx *Context) {
		ctx.bufferSize = size / 4 // float32 = 4 bytes
	}
}

// Context represents the playback context.
type Context struct {
	driverName string
	drv        driver.Driver
	bufferSize int
	mux        *audio.Mixer
}

// NewContext creates a new [Context] with the specified format and options.
// If no driver is specified, the default driver (first one registered) is used.
func NewContext(format afmt.Format, opts ...ContextOption) (*Context, error) {
	ctx := &Context{
		driverName: "",   // Empty string = default driver
		bufferSize: 1024, // Default buffer size
		mux:        audio.NewMixer(nil),
	}

	// Apply options
	for _, opt := range opts {
		opt(ctx)
	}

	ctx.mux.KeepAlive(true)

	if ctx.driverName == "" {
		// Use default driver
		if defaultDriver == nil {
			return nil, fmt.Errorf("playback: no default driver registered")
		}
		ctx.drv = defaultDriver
	} else {
		// Look up and use specified driver
		var ok bool
		driversMu.RLock()
		ctx.drv, ok = drivers[ctx.driverName]
		driversMu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("playback: unknown driver %q (forgotten import?)", ctx.driverName)
		}
	}

	// Init driver
	if err := ctx.drv.Init(format, abufio.NewReaderSize(ctx.mux, ctx.bufferSize)); err != nil {
		return nil, fmt.Errorf("playback: failed to initialize driver %q: %w", ctx.driverName, err)
	}

	return ctx, nil
}

// Close closes the underlying playback driver and the context.
func (ctx *Context) Close() error {
	ctx.mux.Clear()
	return ctx.drv.Close()
}
