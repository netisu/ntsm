package aeno

import (
	"bytes"
	"io"
	"fmt"
	_ "image/jpeg" 
	_ "image/png"
	"aeno"
	"github.com/netisu/ntsm"
)

type LoadedObject struct {
	Object         *aeno.Object
	Emitters       []ntsm.ParticleEmitter
	ParticleSystem *CPUParticleSystem
	Name           string
	GLBData        []byte
	Textures       []aeno.Texture
}

// LoadObject decodes an NTSM stream into an aeno object
func LoadObject(r io.ReadSeeker) (*LoadedObject, error) {
	hdr, glbData, emitters, _, textures, err := ntsm.Decode(r)
	if err != nil {
		return nil, err
	}

	mesh, rootMatrix, err := aeno.LoadGLTFFromReader(bytes.NewReader(glbData))
	if err != nil {
		return nil, err
	}

	loaded := &LoadedObject{
		Object: &aeno.Object{
			Mesh:   mesh,
			Color:  aeno.White,
			Matrix: rootMatrix,
		},
		Emitters: emitters,
		Name:     string(hdr.Name[:]),
		GLBData:  glbData,
		Textures: make([]aeno.Texture, 0),
	}

	for i, t := range textures {
		tex := aeno.TexFromBytes(t.Data)
		if tex != nil {
			loaded.Textures = append(loaded.Textures, tex)
			fmt.Printf("Successfully loaded texture %d: %s\n", i, t.Name)
		} else {
			fmt.Printf("FAILED to load texture %d: %s (Check if data is valid PNG/JPG)\n", i, t.Name)
		}
	}

	if len(loaded.Textures) > 0 {
		loaded.Object.Texture = loaded.Textures[0]
	}

	if len(emitters) > 0 {
		ps := NewCPUParticleSystem(emitters)
		ps.SetTextures(loaded.Textures)
		loaded.ParticleSystem = ps
	}

	return loaded, nil
}
