// Package soxr provides CGo bindings to the SoX Resampler Library, libsoxr.
package soxr

//#cgo !windows pkg-config: soxr
//#cgo windows LDFLAGS: -lsoxr
//#include <soxr.h>
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

// Quality constants for resampling. These map directly to SoX quality presets.
const (
	QualityQuick    uint32 = 0            // QualityQuick represents 'Quick' cubic interpolation.
	QualityLow      uint32 = 1            // QualityLow represents 'Low' 16-bit with larger rolloff.
	QualityMedium   uint32 = 2            // QualityMedium represents 'Medium' 16-bit with medium rolloff.
	QualityHigh     uint32 = Quality20Bit // QualityHigh represents high quality.
	QualityVeryHigh uint32 = Quality28Bit // QualityVeryHigh represents very high quality.

	Quality16Bit uint32 = 3
	Quality20Bit uint32 = 4
	Quality24Bit uint32 = 5
	Quality28Bit uint32 = 6
	Quality32Bit uint32 = 7

	QualityLSR0 uint32 = 8  // QualityLSR0 represents best sinc.
	QualityLSR1 uint32 = 9  // QualityLSR1 represents medium sinc.
	QualityLSR2 uint32 = 10 // QualityLSR2 represents fast sinc.
)

// Version returns the libsoxr version as a string.
func Version() string {
	return C.GoString(C.soxr_version())
}

// Resampler represents a SoX Resampler instance.
type Resampler struct {
	h        C.soxr_t
	r        aio.SampleReader
	isVR     bool
	inRate   freq.Frequency
	outRate  freq.Frequency
	ratio    float64
	channels int
}

func validateInput(r aio.SampleReader, inRate, outRate freq.Frequency, channels int) error {
	if r == nil {
		return errors.New("soxr: reader is nil")
	}
	if inRate <= 0 {
		return errors.New("soxr: invalid input sample rate")
	}
	if outRate <= 0 {
		return errors.New("soxr: invalid output sample rate")
	}
	if channels <= 0 {
		return errors.New("soxr: invalid number of channels")
	}
	return nil
}

// New creates a new [Resampler] with fixed input and output sample rates.
func New(r aio.SampleReader, inRate, outRate freq.Frequency, channels int, quality uint32) (*Resampler, error) {
	if err := validateInput(r, inRate, outRate, channels); err != nil {
		return nil, err
	}

	var h C.soxr_t
	var soxrErr C.soxr_error_t

	ioSpec := C.soxr_io_spec(C.SOXR_FLOAT32_I, C.SOXR_FLOAT32_I)
	qSpec := C.soxr_quality_spec(C.ulong(quality), 0)
	runSpec := C.soxr_runtime_spec(C.uint(runtime.NumCPU())) // use all CPU cores
	h = C.soxr_create(
		C.double(inRate.Hertz()),
		C.double(outRate.Hertz()),
		C.uint(uint(channels)),
		&soxrErr,
		&ioSpec, &qSpec, &runSpec,
	)
	if soxrErr != nil {
		return nil, fmt.Errorf("soxr: %s", C.GoString(soxrErr))
	}

	resampler := &Resampler{
		h:        h,
		r:        r,
		isVR:     false,
		inRate:   inRate,
		outRate:  outRate,
		ratio:    0,
		channels: channels,
	}

	runtime.SetFinalizer(resampler, (*Resampler).Close)

	return resampler, nil
}

// NewWithRatio creates a new [Resampler] in variable-ratio mode with a resampling ratio.
// Unlike [New], this allows dynamic changes to the resampling ratio.
func NewWithRatio(r aio.SampleReader, ratio float64, channels int, quality uint32) (*Resampler, error) {
	// dummy sample rates to make validateInput happy
	if err := validateInput(r, 1, 1, channels); err != nil {
		return nil, err
	}

	var h C.soxr_t
	var soxrErr C.soxr_error_t

	ioSpec := C.soxr_io_spec(C.SOXR_FLOAT32_I, C.SOXR_FLOAT32_I)
	qSpec := C.soxr_quality_spec(C.ulong(quality), C.SOXR_VR) // variable-ratio mode enabled
	runSpec := C.soxr_runtime_spec(C.uint(runtime.NumCPU()))  // use all CPU cores
	h = C.soxr_create(
		C.double(1.0), // dummy input sample rate
		C.double(1.0), // dummy output sample rate
		C.uint(uint(channels)),
		&soxrErr,
		&ioSpec, &qSpec, &runSpec,
	)
	if soxrErr != nil {
		return nil, fmt.Errorf("soxr: %s", C.GoString(soxrErr))
	}

	// set initial ratio
	soxrErr = C.soxr_set_io_ratio(h, C.double(ratio), 0)
	if soxrErr != nil {
		return nil, fmt.Errorf("soxr: %s", C.GoString(soxrErr))
	}

	resampler := &Resampler{
		h:        h,
		r:        r,
		isVR:     true,
		inRate:   0,
		outRate:  0,
		ratio:    ratio,
		channels: channels,
	}
	runtime.SetFinalizer(resampler, (*Resampler).Close)

	return resampler, nil
}

// Close frees the underlying SoX resampler resources.
func (r *Resampler) Close() {
	if r.h != nil {
		runtime.SetFinalizer(r, nil) // prevent double close
		C.soxr_delete(r.h)
		r.h = nil
	}
}

// VariableRatioMode returns whether variable-ratio mode is enabled.
func (r *Resampler) VariableRatioMode() bool {
	return r.isVR
}

// SetRatio updates the resampling ratio in variable-ratio mode.
func (r *Resampler) SetRatio(ratio float64) error {
	if !r.isVR {
		return errors.New("soxr: resampler not in variable-ratio mode")
	}
	if r.h == nil {
		return errors.New("soxr: resampler closed")
	}

	soxrErr := C.soxr_set_io_ratio(r.h, C.double(ratio), 0)
	if soxrErr != nil {
		return fmt.Errorf("soxr: %s", C.GoString(soxrErr))
	}
	r.ratio = ratio
	return nil
}

// Ratio returns the resampling ratio.
func (r *Resampler) Ratio() float64 {
	if r.isVR {
		return r.ratio
	} else {
		return r.outRate.Hertz() / r.inRate.Hertz()
	}
}

// ReadSamples reads samples from the underlying reader, resamples them,
// and writes the output into p. It returns the number of samples written.
func (r *Resampler) ReadSamples(p []float32) (int, error) {
	if r.h == nil {
		return 0, errors.New("soxr: resampler closed")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.channels <= 0 {
		return 0, errors.New("soxr: invalid channel count")
	}
	if len(p)%r.channels != 0 {
		// we'll only fill whole frames; trim the last partial frame.
		p = p[:len(p)-len(p)%r.channels]
		if len(p) == 0 {
			return 0, nil
		}
	}

	ratio := r.Ratio()
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0, errors.New("soxr: invalid resampling ratio")
	}

	outFramesReq := len(p) / r.channels
	if outFramesReq == 0 {
		return 0, nil
	}

	// calculate input frames needed (~= outFramesReq/ratio) and add small headroom
	inFramesNeed := int(math.Ceil(float64(outFramesReq)/ratio)) + 16
	inSamplesNeed := inFramesNeed * r.channels

	in := make([]float32, inSamplesNeed)
	out := make([]float32, outFramesReq*r.channels)

	nInSamples, err := r.r.ReadSamples(in)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("soxr: failed to read samples: %w", err)
	}
	// keep only whole frames
	nInSamples -= nInSamples % r.channels
	if nInSamples == 0 {
		return 0, io.EOF
	}
	in = in[:nInSamples]
	inFrames := nInSamples / r.channels

	var nOutFrames C.size_t
	soxrErr := C.soxr_process(
		r.h,
		C.soxr_in_t(unsafe.Pointer(&in[0])), C.size_t(inFrames), nil,
		C.soxr_out_t(unsafe.Pointer(&out[0])), C.size_t(outFramesReq), &nOutFrames,
	)
	if soxrErr != nil {
		return 0, fmt.Errorf("soxr: %s", C.GoString(soxrErr))
	}

	// convert frames to samples
	nOutSamples := int(nOutFrames) * r.channels
	nOutSamples = min(nOutSamples, len(out))
	copy(p, out[:nOutSamples])
	return nOutSamples, nil
}
