package playback

import (
	"maps"
	"slices"
	"sync"

	"github.com/SladkyCitron/resona/playback/driver"
)

var (
	driversMu     sync.RWMutex
	drivers       = make(map[string]driver.Driver)
	defaultDriver driver.Driver
)

// Register registers a playback driver with the given name.
func Register(name string, drv driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if drv == nil {
		panic("playback: Register driver is nil")
	}
	if _, exists := drivers[name]; exists {
		panic("playback: Register called twice for driver " + name)
	}
	drivers[name] = drv
	if defaultDriver == nil {
		defaultDriver = drv
	}
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	return slices.Sorted(maps.Keys(drivers))
}
