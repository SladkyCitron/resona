package flac

import (
	"errors"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/codec"
	"github.com/SladkyCitron/resona/freq"
	"github.com/mewkiz/flac"
)

const magic = "fLaC"

// Decoder represents the decoder for the FLAC file format.
// It implements codec.Decoder.
type Decoder struct {
	stream   *flac.Stream
	isSeeker bool
	pos      int

	buf []float32
}

// NewDecoder creates a new [Decoder] and decodes the headers.
func NewDecoder(r io.Reader) (_ codec.Decoder, err error) {
	d := &Decoder{}

	rs, ok := r.(io.ReadSeeker)
	d.isSeeker = ok
	if ok {
		d.stream, err = flac.NewSeek(rs)
	} else {
		d.stream, err = flac.New(r)
	}
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Format returns the audio stream format.
func (d *Decoder) Format() afmt.Format {
	return afmt.Format{
		SampleRate:  freq.Frequency(d.stream.Info.SampleRate) * freq.Hertz,
		NumChannels: int(d.stream.Info.NChannels),
	}
}

// SampleFormat returns the sample format.
func (d *Decoder) SampleFormat() afmt.SampleFormat {
	return afmt.SampleFormat{
		BitDepth: int(d.stream.Info.BitsPerSample),
		Encoding: afmt.SampleEncodingInt,
	}
}

// Len returns the total number of frames.
func (d *Decoder) Len() int {
	return int(d.stream.Info.NSamples)
}

// ReadSamples reads float32 samples into p.
// It returns the number of samples read and/or an error.
func (d *Decoder) ReadSamples(p []float32) (int, error) {
	numChannels := int(d.stream.Info.NChannels)
	bitsPerSample := int(d.stream.Info.BitsPerSample)
	scale := float32(int64(1) << (bitsPerSample - 1))

	var n int

	// drain the buffer
	for n < len(p) && len(d.buf) > 0 {
		copied := copy(p[n:], d.buf)
		n += copied
		d.buf = d.buf[copied:]
	}

	// refill the buffer
	for n < len(p) {
		frame, err := d.stream.ParseNext()
		if err != nil {
			if err == io.EOF && n > 0 {
				return n, nil
			}
			return n, err
		}

		numSamples := len(frame.Subframes[0].Samples)
		buf := make([]float32, 0, numSamples*numChannels)

		for i := range numSamples {
			for ch := range numChannels {
				buf = append(buf, float32(frame.Subframes[ch].Samples[i])/scale)
			}
		}

		d.buf = buf

		copied := copy(p[n:], d.buf)
		n += copied
		d.buf = d.buf[copied:]
	}

	d.pos += n / numChannels
	return n, nil
}

// Seek seeks to the specified frame.
// It returns the new offset relative to the start and/or an error.
// It will return an error if the source is not an [io.Seeker].
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	if !d.isSeeker {
		return 0, errors.New("flac: resource does not support seeking")
	}

	// Special case
	if offset == 0 && whence == io.SeekCurrent {
		return int64(d.pos), nil
	}

	var target int
	switch whence {
	case io.SeekStart:
		target = int(offset)
	case io.SeekCurrent:
		target = d.pos + int(offset)
	case io.SeekEnd:
		target = int(d.stream.Info.NSamples) + int(offset)
	default:
		return 0, errors.New("flac: invalid seek whence")
	}

	if target < 0 || target >= int(d.stream.Info.NSamples) {
		return 0, errors.New("flac: seek out of bounds")
	}

	_pos, err := d.stream.Seek(uint64(target))
	if err != nil {
		return 0, err
	}
	d.pos = int(_pos)

	return int64(d.pos), nil
}

func init() {
	codec.RegisterFormat("flac", magic, NewDecoder)
}
