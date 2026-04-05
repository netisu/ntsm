package aeno

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/netisu/aeno"
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
	_, glbData, emitters, _, textures, err := ntsm.Decode(r)
	if err != nil {
		return nil, err
	}

	mesh, err := aeno.LoadGLTFFromReader(bytes.NewReader(glbData))
	if err != nil {
		return nil, err
	}

	loaded := &LoadedObject{
		Object: &aeno.Object{
			Mesh:   mesh,
			Color:  aeno.Transparent,
			Matrix: aeno.Identity(),
		},
		Emitters: emitters,
	}

	for _, texData := range textures {
		tmpPath := filepath.Join(os.TempDir(), texData.Name)
		if err := os.WriteFile(tmpPath, texData.Data, 0644); err == nil {
			tex := aeno.LoadTextureFromURL(tmpPath)
			loaded.Textures = append(loaded.Textures, tex)
			// Clean up the temp file after loading it into memory
			os.Remove(tmpPath)
		}
	}

	if len(loaded.Textures) > 0 {
		loaded.Object.Texture = loaded.Textures[0]
	}

	if len(emitters) > 0 {
		loaded.ParticleSystem = NewCPUParticleSystem(emitters)
	}

	return loaded, nil
}
