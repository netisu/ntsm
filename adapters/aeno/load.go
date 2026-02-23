package aeno

import (
	"bytes"
	"io"

	"github.com/netisu/aeno"
	"github.com/netisu/ntsm"
)

type LoadedObject struct {
	*aeno.Object
	Emitters []ntsm.ParticleEmitter
}

// LoadObject decodes an NTSM stream into an aeno object
func LoadObject(r io.Reader) (*LoadedObject, error) {
	hdr, glbData, emitters, err := ntsm.Decode(r)
	if err != nil {
		return nil, err
	}

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
	}, nil
}
