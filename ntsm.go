package ntsm

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	_              [3]byte // Padding
	GLBOffset      uint32
	GLBSize        uint32
	ParticleOffset uint32
	ParticleSize   uint32
	TextureCount   uint32
	TextureOffset  uint32
	ScriptCount    uint32
	ScriptOffset   uint32
	_              [20]byte // Padding
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

	MaxParticles uint32
	SpawnCount   uint32
	Reserved     [8]byte // Padding to reach 128 bytes exactly
}

// Script represents a decoded Luau script
type Script struct {
	Name string
	Code string
}

// ScriptTableEntry defines how scripts are located in the binary
type ScriptTableEntry struct {
	Name   [64]byte
	Offset uint32
	Size   uint32
}

type Texture struct {
	Name string
	Data []byte
}

// TextureTableEntry defines how textures are located in the binary
type TextureTableEntry struct {
	Name   [64]byte
	Size   uint32
	Offset uint32
}

// Decode reads an NTSM file and returns header, GLB bytes, and emitters
func Decode(r io.ReadSeeker) (*Header, []byte, []ParticleEmitter, []Script, []Texture, error) {
	var hdr Header
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	_, err := r.Seek(int64(hdr.GLBOffset), io.SeekStart)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	glbData := make([]byte, hdr.GLBSize)
	if _, err := io.ReadFull(r, glbData); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var emitters []ParticleEmitter
	if hdr.ParticleSize > 0 {
		_, _ = r.Seek(int64(hdr.ParticleOffset), io.SeekStart)
		emitterCount := hdr.ParticleSize / uint32(binary.Size(ParticleEmitter{}))
		emitters = make([]ParticleEmitter, emitterCount)
		binary.Read(r, binary.LittleEndian, &emitters)
	}

	var scripts []Script
	if hdr.ScriptCount > 0 {
		_, _ = r.Seek(int64(hdr.ScriptOffset), io.SeekStart)
		table := make([]ScriptTableEntry, hdr.ScriptCount)
		binary.Read(r, binary.LittleEndian, &table)

		for _, entry := range table {
			_, _ = r.Seek(int64(entry.Offset), io.SeekStart)
			codeData := make([]byte, entry.Size)
			io.ReadFull(r, codeData)
			nameStr := string(bytes.TrimRight(entry.Name[:], "\x00"))
			scripts = append(scripts, Script{Name: nameStr, Code: string(codeData)})
		}
	}

	table := make([]TextureTableEntry, hdr.TextureCount)
	_, err = r.Seek(int64(hdr.TextureOffset), io.SeekStart)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &table); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	textures := make([]Texture, hdr.TextureCount)
	for i, entry := range table {
		_, err = r.Seek(int64(entry.Offset), io.SeekStart)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		data := make([]byte, entry.Size)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, nil, nil, nil, nil, err
		}

		nameStr := string(bytes.TrimRight(entry.Name[:], "\x00"))
		textures[i] = Texture{
			Name: nameStr,
			Data: data,
		}
	}

	fmt.Printf("Seeking to TextureOffset: %d\n", hdr.TextureOffset)

	return &hdr, glbData, emitters, scripts, textures, nil
}
