package wav

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/codec"
	"github.com/SladkyCitron/resona/codec/mp3"
	"github.com/SladkyCitron/resona/codec/wav/internal/riff"
	"github.com/SladkyCitron/resona/encoding/dfpwm"
	"github.com/SladkyCitron/resona/encoding/g711"
	"github.com/SladkyCitron/resona/encoding/pcm"
	"github.com/SladkyCitron/resona/freq"
)

// Chunk IDs for the WAVE file format.
var (
	WaveID riff.FourCC = riff.FourCC{'W', 'A', 'V', 'E'}
	FmtID  riff.FourCC = riff.FourCC{'f', 'm', 't', ' '}
	DataID riff.FourCC = riff.FourCC{'d', 'a', 't', 'a'}
)

const magic string = "RIFF????WAVE"

// Decoder represents the decoder for the WAVE file format.
// It implements codec.Decoder.
type Decoder struct {
	riffR *riff.Reader

	// AudioFormat is the WAVE audio format.
	AudioFormat uint16

	numChannels   uint16
	sampleRate    uint32
	bytesPerSec   uint32
	bytesPerBlock uint16
	bitsPerSample uint16

	// ChannelMask is the speaker position mask.
	// It's only valid if the audio format is WAVE_FORMAT_EXTENSIBLE (0xFFFE).
	ChannelMask uint32

	// SubformatGUID is the WAVEX subformat GUID.
	// It's only valid if the audio format is WAVE_FORMAT_EXTENSIBLE (0xFFFE).
	SubformatGUID GUID

	dataChunk *riff.Chunk
	dataRead  int

	dec aio.SampleReader
}

// NewDecoder creates a new [Decoder] and decodes the headers.
func NewDecoder(r io.Reader) (_ codec.Decoder, err error) {
	d := &Decoder{
		SubformatGUID: GUID{},
	}

	var id riff.FourCC
	id, d.riffR, err = riff.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RIFF stream: %w", err)
	}

	if !bytes.Equal(id[:], WaveID[:]) {
		return nil, fmt.Errorf("invalid WAVE header: %v", id)
	}

	if err := d.parseFmt(); err != nil {
		return nil, fmt.Errorf("failed to parse fmt chunk: %w", err)
	}

	for {
		chunk, err := d.riffR.NextChunk()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		switch {
		case bytes.Equal(chunk.ID[:], DataID[:]):
			d.dataChunk = chunk
			if err := d.ensureAudioDecoder(); err != nil {
				return nil, err
			}
			return d, nil // success
		default:
			// Skip unknown chunk
			_, _ = io.Copy(io.Discard, chunk.Reader)
		}
	}
	if d.dataChunk == nil {
		return nil, fmt.Errorf("invalid or missing data chunk")
	}
	return d, nil
}

// parseFmt reads and parses the "fmt " chunk.
func (d *Decoder) parseFmt() error {
	chunk, err := d.riffR.NextChunk()
	if err != nil {
		return err
	}

	if !bytes.Equal(chunk.ID[:], FmtID[:]) {
		return fmt.Errorf("invalid or missing fmt chunk")
	}

	if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.AudioFormat); err != nil {
		return fmt.Errorf("failed to read audio format: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.numChannels); err != nil {
		return fmt.Errorf("failed to read number of channels: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.sampleRate); err != nil {
		return fmt.Errorf("failed to read sample rate: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.bytesPerSec); err != nil {
		return fmt.Errorf("failed to read bytes pre second: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.bytesPerBlock); err != nil {
		return fmt.Errorf("failed to read bytes per block: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.bitsPerSample); err != nil {
		return fmt.Errorf("failed to read bits per sample: %w", err)
	}

	// WAVEX
	// https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ksmedia/ns-ksmedia-waveformatextensible
	if d.AudioFormat == FormatWAVEX {
		_, _ = io.CopyN(io.Discard, chunk.Reader, 2) // skip cbSize

		// valid bits per sample
		// I think this is how you parse it??? It's actually a C union but we don't have unions in Go
		if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.bitsPerSample); err != nil {
			return fmt.Errorf("failed to read valid bits per sample: %w", err)
		}

		if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.ChannelMask); err != nil {
			return fmt.Errorf("failed to read channel mask: %w", err)
		}

		if err := binary.Read(chunk.Reader, binary.LittleEndian, &d.SubformatGUID); err != nil {
			return fmt.Errorf("failed to read subformat GUID: %w", err)
		}
	}

	return nil
}

func (d *Decoder) ensureAudioDecoder() error {
	switch d.AudioFormat {
	case FormatInt, FormatFloat:
		d.dec = pcm.NewDecoder(d.dataChunk.Reader, d.SampleFormat())
	case FormatAlaw:
		d.dec = g711.NewAlawDecoder(d.dataChunk.Reader)
	case FormatUlaw:
		d.dec = g711.NewUlawDecoder(d.dataChunk.Reader)
	case FormatMP3:
		var err error
		d.dec, err = mp3.NewDecoder(d.dataChunk.Reader)
		if err != nil {
			return err
		}
	case FormatWAVEX:
		switch d.SubformatGUID {
		case GuidInt, GuidFloat:
			d.dec = pcm.NewDecoder(d.dataChunk.Reader, d.SampleFormat())
		case GuidAlaw:
			d.dec = g711.NewAlawDecoder(d.dataChunk.Reader)
		case GuidUlaw:
			d.dec = g711.NewUlawDecoder(d.dataChunk.Reader)
		case GuidMP3:
			var err error
			d.dec, err = mp3.NewDecoder(d.dataChunk.Reader)
			if err != nil {
				return err
			}
		case GuidDFPWM:
			d.dec = dfpwm.NewDecoder(d.dataChunk.Reader)
		default:
			return fmt.Errorf("unknown subformat GUID: %v", d.SubformatGUID)
		}
	default:
		return fmt.Errorf("unknown audio format: %v", d.AudioFormat)
	}

	return nil
}

// Bitrate returns the bitrate of the audio stream in bytes per second.
func (d *Decoder) Bitrate() int {
	if bitrater, ok := d.dec.(codec.Bitrater); ok {
		return bitrater.Bitrate()
	}

	return int(d.bytesPerSec) * 8
}

// Format returns the audio stream format.
func (d *Decoder) Format() afmt.Format {
	return afmt.Format{
		SampleRate:  freq.Frequency(d.sampleRate) * freq.Hertz,
		NumChannels: int(d.numChannels),
	}
}

// SampleFormat returns the sample format.
func (d *Decoder) SampleFormat() afmt.SampleFormat {
	f := afmt.SampleFormat{
		BitDepth: int(d.bitsPerSample),
		Endian:   binary.LittleEndian,
	}

	switch d.AudioFormat {
	case FormatInt:
		f.Encoding = afmt.SampleEncodingInt
	case FormatFloat:
		f.Encoding = afmt.SampleEncodingFloat
	case FormatAlaw, FormatUlaw:
		f.Encoding = afmt.SampleEncodingUint
		f.Endian = nil
	case FormatWAVEX:
		switch d.SubformatGUID {
		case GuidInt:
			f.Encoding = afmt.SampleEncodingInt
		case GuidFloat:
			f.Encoding = afmt.SampleEncodingFloat
		case GuidAlaw, GuidUlaw:
			f.Encoding = afmt.SampleEncodingUint
			f.Endian = nil
		case GuidDFPWM:
			f.BitDepth = 1
			f.Encoding = afmt.SampleEncodingUint
			f.Endian = nil
		}
	}
	if d.bitsPerSample == 8 {
		f.Encoding = afmt.SampleEncodingUint // 8-bit is always unsigned
		f.Endian = nil
	}

	return f
}

// ReadSamples reads float32 samples from the data chunk into p.
// It returns the number of samples read and/or an error.
func (d *Decoder) ReadSamples(p []float32) (n int, err error) {
	n, err = d.dec.ReadSamples(p)
	d.dataRead += n
	return
}

// Len returns the total number of frames.
func (d *Decoder) Len() int {
	frameSize := int(d.bytesPerBlock)
	if frameSize == 0 {
		return 0
	}
	return d.dataChunk.Len / frameSize
}

// Seek seeks to the specified frame.
// It returns the new offset relative to the start and/or an error.
// It will return an error if the source is not an [io.Seeker].
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	// Disable seeking for RIFF MP3s
	// for some reason seeking/position is broken in RIFF MP3
	if _, ok := d.dec.(*mp3.Decoder); ok {
		return 0, fmt.Errorf("wav: seeking not supported for MP3-encoded WAV files (0x55)")
	}

	// Special case
	if offset == 0 && whence == io.SeekCurrent {
		return int64(d.dataRead) / int64(d.numChannels), nil
	}

	frameSize := int64(d.bytesPerBlock)

	totalFrames := int64(d.dataChunk.Len) / frameSize

	var targetFrame int64
	switch whence {
	case io.SeekStart:
		targetFrame = offset
	case io.SeekCurrent:
		targetFrame = int64(d.dataRead) + offset
	case io.SeekEnd:
		targetFrame = totalFrames + offset
	default:
		return 0, fmt.Errorf("wav: invalid seek whence")
	}

	if targetFrame < 0 || targetFrame > totalFrames {
		return 0, fmt.Errorf("wav: seek out of bounds")
	}

	byteOffset := targetFrame * frameSize

	_, err := d.dataChunk.Reader.Seek(byteOffset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("wav: failed to seek: %w", err)
	}

	d.dataRead = int(byteOffset) / int(frameSize)
	return targetFrame, nil
}

func init() {
	codec.RegisterFormat("wav", magic, NewDecoder)
}
