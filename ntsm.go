package ntsm

import (
	"encoding/binary"
	"io"
)

const (
	Magic      = "NTSM"
	Version    = 1
	HeaderSize = 192
)

type Header struct {
	Magic          [4]byte
	Version        uint32
	Name           [128]byte
	Flags          uint8
	GLBOffset      uint32
	GLBSize        uint32
	ParticleOffset uint32
	ParticleSize   uint32
}

type ParticleEmitter struct {
	Position         [3]float32
	Direction        [3]float32
	SpreadAngle      float32
	EmissionRate     float32
	ParticleLifetime float32
	StartSize        float32
	EndSize          float32
	StartColor       [4]float32
	EndColor         [4]float32
	VelocityMin      [3]float32
	VelocityMax      [3]float32
	Gravity          float32
	TextureIndex     int32
	BlendMode        uint8
	Loop             uint8
	_                [2]byte
}

// Decode reads an NTSM file and returns header, GLB bytes, and emitters
func Decode(r io.Reader) (*Header, []byte, []ParticleEmitter, error) {
	var hdr Header
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return nil, nil, nil, err
	}

	padding := make([]byte, HeaderSize-192)
	io.ReadFull(r, padding)
	glbData := make([]byte, hdr.GLBSize)
	if _, err := io.ReadFull(r, glbData); err != nil {
		return nil, nil, nil, err
	}

	var emitters []ParticleEmitter
	if hdr.ParticleSize > 0 && (hdr.Flags&0x01) != 0 {
		count := hdr.ParticleSize / 128
		emitters = make([]ParticleEmitter, count)
		for i := range emitters {
			if err := binary.Read(r, binary.LittleEndian, &emitters[i]); err != nil {
				return nil, nil, nil, err
			}
		}
	}

	return &hdr, glbData, emitters, nil
}
