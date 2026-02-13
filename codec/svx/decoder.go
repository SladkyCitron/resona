package svx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/codec"
	"github.com/SladkyCitron/resona/codec/internal/iff"
	"github.com/SladkyCitron/resona/encoding/pcm"
	"github.com/SladkyCitron/resona/freq"
)

// Chunk IDs.
var (
	EightSVXID   iff.FourCC = iff.FourCC{'8', 'S', 'V', 'X'}
	SixteenSVXID iff.FourCC = iff.FourCC{'1', '6', 'S', 'V'}

	VHDRID iff.FourCC = iff.FourCC{'V', 'H', 'D', 'R'}
	/*
		NameID      iff.FourCC = iff.FourCC{'N', 'A', 'M', 'E'}
		CopyrightID iff.FourCC = iff.FourCC{'(', 'c', ')', ' '}
		AuthorID    iff.FourCC = iff.FourCC{'A', 'U', 'T', 'H'}
		AnnoID      iff.FourCC = iff.FourCC{'A', 'N', 'N', 'O'}
	*/
	BodyID iff.FourCC = iff.FourCC{'B', 'O', 'D', 'Y'}
)

// Decoder represents the decoder for the Amiga IFF/8SVX/16SVX file format.
// It implements codec.Decoder.
type Decoder struct {
	iffR *iff.Reader

	bitDepth uint8

	oneShotLength     uint32
	loopLength        uint32
	numLoops          uint32
	sampleRate        uint16
	NumOctaves        uint8
	CompressionMethod uint8
	Volume            Fixed16_16

	bodyChunk *iff.Chunk
	dataRead  int

	pcmDec aio.SampleReader
}

// NewDecoder creates a new [Decoder] and decodes the headers.
func NewDecoder(r io.Reader) (_ codec.Decoder, err error) {
	d := &Decoder{}

	var id iff.FourCC
	id, d.iffR, err = iff.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IFF stream: %w", err)
	}

	switch {
	case bytes.Equal(id[:], EightSVXID[:]):
		d.bitDepth = 8
	case bytes.Equal(id[:], SixteenSVXID[:]):
		d.bitDepth = 16
	default:
		return nil, fmt.Errorf("invalid header")
	}

	if err := d.parseVHDR(); err != nil {
		return nil, fmt.Errorf("failed to parse VHDR chunk: %w", err)
	}

	for {
		chunk, err := d.iffR.NextChunk()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		switch {
		case bytes.Equal(chunk.ID[:], BodyID[:]):
			d.bodyChunk = chunk
			d.pcmDec = pcm.NewDecoder(d.bodyChunk.Reader, d.SampleFormat())
			return d, nil
		default:
			_, _ = io.Copy(io.Discard, chunk.Reader)
		}
	}
	if d.bodyChunk == nil {
		return nil, fmt.Errorf("invalid or missing BODY chunk")
	}
	return d, nil
}

func (d *Decoder) parseVHDR() error {
	chunk, err := d.iffR.NextChunk()
	if err != nil {
		return err
	}

	if !bytes.Equal(chunk.ID[:], VHDRID[:]) {
		return errors.New("invalid or missing VHDR header")
	}

	if err := binary.Read(chunk.Reader, binary.BigEndian, &d.oneShotLength); err != nil {
		return fmt.Errorf("failed to read one-shot length: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.BigEndian, &d.loopLength); err != nil {
		return fmt.Errorf("failed to read loop length: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.BigEndian, &d.numLoops); err != nil {
		return fmt.Errorf("failed to read number of loops: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.BigEndian, &d.sampleRate); err != nil {
		return fmt.Errorf("failed to read sample rate: %w", err)
	}
	if err := binary.Read(chunk.Reader, binary.BigEndian, &d.Volume); err != nil {
		return fmt.Errorf("failed to read volume: %w", err)
	}

	return nil
}

// Format returns the audio stream format.
func (d *Decoder) Format() afmt.Format {
	return afmt.Format{
		SampleRate:  freq.Frequency(d.sampleRate) * freq.Hertz,
		NumChannels: 1, // always mono
	}
}

// SampleFormat returns the sample format.
func (d *Decoder) SampleFormat() afmt.SampleFormat {
	return afmt.SampleFormat{
		BitDepth: int(d.bitDepth),
		Encoding: afmt.SampleEncodingInt,
		Endian:   binary.BigEndian,
	}
}

// ReadSamples reads float32 samples from the data chunk into p.
// It returns the number of samples read and/or an error.
func (d *Decoder) ReadSamples(p []float32) (n int, err error) {
	n, err = d.pcmDec.ReadSamples(p)
	d.dataRead += n * int(d.bitDepth/8)
	return n, err
}

// Len returns the total number of frames (mono samples).
func (d *Decoder) Len() int {
	frameSize := int(d.bitDepth / 8)
	if frameSize == 0 {
		return 0
	}
	return d.bodyChunk.Len / frameSize
}

// Seek seeks to the specified frame (mono sample).
// It returns the new offset relative to the start and/or an error.
// It will return an error if the source is not an [io.Seeker].
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	// Special case
	if offset == 0 && whence == io.SeekCurrent {
		return int64(d.dataRead) / int64(d.SampleFormat().BytesPerSample()), nil
	}

	frameSize := int64(d.bitDepth / 8)
	totalFrames := int64(d.bodyChunk.Len) / frameSize

	var targetFrame int64
	switch whence {
	case io.SeekStart:
		targetFrame = offset
	case io.SeekCurrent:
		targetFrame = int64(d.dataRead)/frameSize + offset
	case io.SeekEnd:
		targetFrame = totalFrames + offset
	default:
		return 0, fmt.Errorf("svx: invalid seek whence")
	}

	if targetFrame < 0 || targetFrame > totalFrames {
		return 0, fmt.Errorf("svx: seek out of bounds")
	}

	byteOffset := targetFrame * frameSize

	_, err := d.bodyChunk.Reader.Seek(byteOffset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("svx: failed to seek: %w", err)
	}

	d.dataRead = int(byteOffset)
	return targetFrame, nil
}

// OneShotLen returns the one shot length in frames (mono samples).
func (d *Decoder) OneShotLen() int {
	frameSize := int(d.bitDepth / 8)
	if frameSize == 0 {
		return 0
	}
	return int(d.oneShotLength) / frameSize
}

// LoopLen returns the loop length in frames (mono samples).
func (d *Decoder) LoopLen() int {
	frameSize := int(d.bitDepth / 8)
	if frameSize == 0 {
		return 0
	}
	return int(d.loopLength) / frameSize
}

// NumLoops returns the number of loops.
func (d *Decoder) NumLoops() int {
	return int(d.numLoops)
}

func init() {
	codec.RegisterFormat("8svx", "FORM????8SVX", NewDecoder)
	codec.RegisterFormat("16svx", "FORM????16SV", NewDecoder)
}
