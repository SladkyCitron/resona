// Package libsamplerate provides CGo bindings for the libsamplerate (Secret Rabbit Code) audio resampling library.
package libsamplerate

/*
#cgo !windows pkg-config: samplerate
#cgo windows LDFLAGS: -lsamplerate

#include <samplerate.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"unsafe"

	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/freq"
)

// Quality constants for resampling. These map directly to [libsamplerate converter types].
//
// [libsamplerate converter types]: https://libsndfile.github.io/libsamplerate/api_misc.html#converters
const (
	QualitySincBest      int = C.SRC_SINC_BEST_QUALITY
	QualitySincMedium    int = C.SRC_SINC_MEDIUM_QUALITY
	QualitySincFastest   int = C.SRC_SINC_FASTEST
	QualityZeroOrderHold int = C.SRC_ZERO_ORDER_HOLD
	QualityLinear        int = C.SRC_LINEAR
)

// Version returns the libsamplerate version as a string.
func Version() string {
	return C.GoString(C.src_get_version())
}

// Resampler represents a libsamplerate instance.
type Resampler struct {
	srcState *C.SRC_STATE
	srcData  C.SRC_DATA
	r        aio.SampleReader
	channels int
	ratio    float64

	// C-allocated buffers to make "cgo argument has Go pointer to unpinned Go pointer" happy
	inBuf     *C.float // data_in
	inBufCap  int      // samples
	outBuf    *C.float // data_out
	outBufCap int      // samples
}

// New creates a new [Resampler] with fixed input and output sample rates.
func New(r aio.SampleReader, inRate, outRate freq.Frequency, channels int, quality int) (*Resampler, error) {
	var err C.int
	srcState := C.src_new(C.int(quality), C.int(channels), &err)
	if err != 0 {
		return nil, fmt.Errorf("libsamplerate: %s", C.GoString(C.src_strerror(err)))
	}

	var srcData C.SRC_DATA
	srcData.src_ratio = C.double(outRate.Hertz() / inRate.Hertz())
	srcData.end_of_input = C.int(0)

	resampler := &Resampler{
		srcState: srcState,
		srcData:  srcData,
		r:        r,
		channels: channels,
		ratio:    float64(srcData.src_ratio),
	}

	runtime.SetFinalizer(resampler, (*Resampler).Close)

	return resampler, nil
}

// NewWithRatio creates a new [Resampler] with a resampling ratio.
func NewWithRatio(r aio.SampleReader, ratio float64, channels int, quality int) (*Resampler, error) {
	var err C.int
	srcState := C.src_new(C.int(quality), C.int(channels), &err)
	if err != 0 {
		return nil, fmt.Errorf("libsamplerate: %s", C.GoString(C.src_strerror(err)))
	}

	var srcData C.SRC_DATA
	srcData.src_ratio = C.double(1.0 / ratio)
	srcData.end_of_input = C.int(0)

	resampler := &Resampler{
		srcState: srcState,
		srcData:  srcData,
		r:        r,
		channels: channels,
		ratio:    ratio,
	}

	runtime.SetFinalizer(resampler, (*Resampler).Close)

	return resampler, nil
}

// Close frees the underlying resampler resources.
func (r *Resampler) Close() {
	if r == nil {
		return
	}

	// free C buffers first
	if r.inBuf != nil {
		C.free(unsafe.Pointer(r.inBuf))
		r.inBuf = nil
		r.inBufCap = 0
	}
	if r.outBuf != nil {
		C.free(unsafe.Pointer(r.outBuf))
		r.outBuf = nil
		r.outBufCap = 0
	}
	if r.srcState != nil {
		runtime.SetFinalizer(r, nil) // prevent double close
		C.src_delete(r.srcState)
		r.srcState = nil
	}
}

// SetRatio updates the resampling ratio.
func (r *Resampler) SetRatio(ratio float64) error {
	if r.srcState == nil {
		return errors.New("libsamplerate: resampler closed")
	}
	err := C.src_set_ratio(r.srcState, C.double(1.0/ratio))
	if err != 0 {
		return fmt.Errorf("libsamplerate: %s", C.GoString(C.src_strerror(err)))
	}
	r.srcData.src_ratio = C.double(1.0 / ratio)
	r.ratio = ratio
	return nil
}

// Ratio returns the resampling ratio.
func (r *Resampler) Ratio() float64 {
	return r.ratio
}

// ensureCFloatCap ensures a C float buffer has at least n samples capacity.
func ensureCFloatCap(p **C.float, capPtr *int, n int) {
	if *capPtr >= n {
		return
	}
	size := C.size_t(n) * C.size_t(unsafe.Sizeof(*(*C.float)(nil)))
	if *p == nil {
		*p = (*C.float)(C.malloc(size))
	} else {
		*p = (*C.float)(C.realloc(unsafe.Pointer(*p), size))
	}
	*capPtr = n
}

// ReadSamples reads samples from the underlying reader, resamples them,
// and writes the output into p. It returns the number of samples written.
func (r *Resampler) ReadSamples(p []float32) (int, error) {
	if r.srcState == nil {
		return 0, errors.New("libsamplerate: resampler closed")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.channels <= 0 {
		return 0, errors.New("libsamplerate: invalid channel count")
	}
	if len(p)%r.channels != 0 {
		// we'll only fill whole frames; trim the last partial frame.
		p = p[:len(p)-len(p)%r.channels]
		if len(p) == 0 {
			return 0, nil
		}
	}

	ratio := float64(r.srcData.src_ratio)
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0, errors.New("libsamplerate: invalid resampling ratio")
	}

	outFramesReq := len(p) / r.channels
	if outFramesReq == 0 {
		return 0, nil
	}

	// calculate input frames needed (~= outFramesReq/ratio) and add small headroom
	inFramesNeed := int(math.Ceil(float64(outFramesReq)/ratio)) + 16
	inSamplesNeed := inFramesNeed * r.channels

	in := make([]float32, inSamplesNeed)

	nInSamples, err := r.r.ReadSamples(in)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("libsamplerate: failed to read samples: %w", err)
	}
	// keep only whole frames
	nInSamples -= nInSamples % r.channels
	if nInSamples == 0 {
		r.srcData.end_of_input = C.int(1)
		return 0, io.EOF
	}
	in = in[:nInSamples]
	inFrames := nInSamples / r.channels

	// ensure C buffers
	ensureCFloatCap(&r.inBuf, &r.inBufCap, nInSamples)
	ensureCFloatCap(&r.outBuf, &r.outBufCap, outFramesReq*r.channels)

	// view C memory as Go slices (backed by C heap).
	inC := unsafe.Slice(r.inBuf, nInSamples)
	outC := unsafe.Slice(r.outBuf, outFramesReq*r.channels)

	// convert float32 -> C.float (float32).
	for i := 0; i < nInSamples; i++ {
		inC[i] = C.float(in[i])
	}

	// Set up SRC_DATA
	r.srcData.data_in = r.inBuf
	r.srcData.data_out = r.outBuf
	r.srcData.input_frames = C.long(inFrames)
	r.srcData.output_frames = C.long(outFramesReq)

	cErr := C.src_process(r.srcState, &r.srcData)
	if cErr != 0 {
		return 0, fmt.Errorf("libsamplerate: %s", C.GoString(C.src_strerror(cErr)))
	}

	// collect output
	nOutFrames := int(r.srcData.output_frames_gen)
	nOutSamples := nOutFrames * r.channels
	for i := 0; i < nOutSamples; i++ {
		p[i] = float32(outC[i])
	}

	// reset end_of_input if we had more data earlier
	r.srcData.end_of_input = C.int(0)

	return nOutSamples, nil
}
