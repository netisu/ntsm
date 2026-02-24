package aeno

import (
	"bytes"
	"io"

	"github.com/netisu/aeno"
	"github.com/netisu/ntsm"
)

type LoadedObject struct {
	Object *aeno.Object
	Emitters []ntsm.ParticleEmitter
	Name     string
	GLBData []byte
}

// LoadObject decodes an NTSM stream into an aeno object
func LoadObject(r io.Reader) (*LoadedObject, error) {
	hdr, glbData, emitters, err := ntsm.Decode(r)
	if err != nil {
		return nil, err
	}

	itemName := string(hdr.Name[:])
	itemName = itemName[:len(itemName)-1]

	mesh, err := aeno.LoadGLTFFromReader(bytes.NewReader(glbData))
	if err != nil {
		return nil, err
	}

	return &LoadedObject{
		Object: &aeno.Object{
			Mesh:   mesh,
			Color:  aeno.Transparent,
			Matrix: aeno.Identity(),
		},
		Emitters: emitters,
		Name:     itemName,
		GLBData: glbData,
	}, nil
}
