package mp3

import (
	"errors"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/codec"
	"github.com/SladkyCitron/resona/codec/mp3/internal/consts"
	"github.com/SladkyCitron/resona/codec/mp3/internal/frame"
	"github.com/SladkyCitron/resona/codec/mp3/internal/frameheader"
	"github.com/SladkyCitron/resona/freq"
)

// Decoder represents the decoder for the MP3 file format.
// It implements codec.Decoder.
type Decoder struct {
	source        *source
	sampleRate    int
	length        int64
	frameStarts   []int64
	buf           []float32
	frame         *frame.Frame
	pos           int64
	bytesPerFrame int64
}

// NewDecoder creates a new [Decoder] and decodes the headers.
func NewDecoder(r io.Reader) (_ codec.Decoder, err error) {
	s := &source{
		reader: r,
	}
	d := &Decoder{
		source: s,
		length: invalidLength,
	}

	if err := s.skipTags(); err != nil {
		return nil, err
	}
	// TODO: Is readFrame here really needed?
	if err := d.readFrame(); err != nil {
		return nil, err
	}
	freq, err := d.frame.SamplingFrequency()
	if err != nil {
		return nil, err
	}
	d.sampleRate = freq

	if err := d.ensureFrameStartsAndLength(); err != nil {
		return nil, err
	}

	return d, nil
}

// Bitrate returns the bitrate of the audio stream in bytes per second.
func (d *Decoder) Bitrate() int {
	return d.frame.Bitrate()
}

// Format returns the audio stream format. Audio is always stereo.
func (d *Decoder) Format() afmt.Format {
	return afmt.Format{
		SampleRate:  freq.Frequency(d.sampleRate) * freq.Hertz,
		NumChannels: 2,
	}
}

// SampleFormat returns the sample format that samples are being decoded to internally.
// Note that this isn't actually the audio stream's sample format, as it's compressed.
func (d *Decoder) SampleFormat() afmt.SampleFormat {
	return afmt.SampleFormat{
		BitDepth: 32,
		Encoding: afmt.SampleEncodingFloat,
	}
}

func (d *Decoder) ensureFrameStartsAndLength() error {
	if d.length != invalidLength {
		return nil
	}

	if _, ok := d.source.reader.(io.Seeker); !ok {
		return nil
	}

	// Keep the current position.
	pos, err := d.source.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if err := d.source.rewind(); err != nil {
		return err
	}

	if err := d.source.skipTags(); err != nil {
		return err
	}
	l := int64(0)
	for {
		h, pos, err := frameheader.Read(d.source, d.source.pos)
		if err != nil {
			if err == io.EOF {
				break
			}
			if _, ok := err.(*consts.UnexpectedEOF); ok {
				// TODO: Log here?
				break
			}
			return err
		}
		d.frameStarts = append(d.frameStarts, pos)
		d.bytesPerFrame = int64(h.BytesPerFrame())
		l += d.bytesPerFrame

		framesize, err := h.FrameSize()
		if err != nil {
			return err
		}
		buf := make([]byte, framesize-4)
		if _, err := d.source.ReadFull(buf); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	d.length = l

	if _, err := d.source.Seek(pos, io.SeekStart); err != nil {
		return err
	}
	return nil
}

const invalidLength = -1

func (d *Decoder) readFrame() error {
	var err error
	d.frame, _, err = frame.Read(d.source, d.source.pos, d.frame)
	if err != nil {
		if err == io.EOF {
			return io.EOF
		}
		if _, ok := err.(*consts.UnexpectedEOF); ok {
			// TODO: Log here?
			return io.EOF
		}
		return err
	}
	d.buf = append(d.buf, d.frame.Decode()...)
	return nil
}

// ReadSamples reads float32 samples into p.
// It returns the number of samples read and/or an error.
func (d *Decoder) ReadSamples(p []float32) (int, error) {
	for len(d.buf) == 0 {
		if err := d.readFrame(); err != nil {
			return 0, err
		}
	}
	n := copy(p, d.buf)
	d.buf = d.buf[n:]
	d.pos += int64(n)

	return n, nil
}

// Len returns the total number of frames.
func (d *Decoder) Len() int {
	return int(d.length) / 4
}

// Seek seeks to the specified frame.
// It returns the new offset relative to the start and/or an error.
// It will return an error if the source is not an [io.Seeker].
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	offset *= 4

	if offset == 0 && whence == io.SeekCurrent {
		// Handle the special case of asking for the current position specially.
		return d.pos / 2, nil
	}

	npos := int64(0)
	switch whence {
	case io.SeekStart:
		npos = offset
	case io.SeekCurrent:
		npos = d.pos + offset
	case io.SeekEnd:
		npos = d.length + offset
	default:
		return 0, errors.New("mp3: invalid whence")
	}
	d.pos = npos
	d.buf = nil
	d.frame = nil
	f := d.pos / d.bytesPerFrame
	// If the frame is not first, read the previous ahead of reading that
	// because the previous frame can affect the targeted frame.
	if f > 0 {
		f--
		if _, err := d.source.Seek(d.frameStarts[f], 0); err != nil {
			return 0, err
		}
		if err := d.readFrame(); err != nil {
			return 0, err
		}
		if err := d.readFrame(); err != nil {
			return 0, err
		}
		d.buf = d.buf[d.bytesPerFrame+(d.pos%d.bytesPerFrame):]
	} else {
		if _, err := d.source.Seek(d.frameStarts[f], 0); err != nil {
			return 0, err
		}
		if err := d.readFrame(); err != nil {
			return 0, err
		}
		d.buf = d.buf[d.pos:]
	}
	return npos / 4, nil
}

func init() {
	// Without ID3v2
	codec.RegisterFormat("mp3", string([]byte{0xFF, 0xFB}), NewDecoder)
	codec.RegisterFormat("mp3", string([]byte{0xFF, 0xF3}), NewDecoder)
	codec.RegisterFormat("mp3", string([]byte{0xFF, 0xF2}), NewDecoder)

	// With ID3v2
	codec.RegisterFormat("mp3", string([]byte{0x49, 0x44, 0x33}), NewDecoder)
}
