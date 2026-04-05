package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/netisu/aeno"
	aenoAdapter "github.com/netisu/ntsm/adapters/aeno"
)

const (
	Dimensions = 512
	Scale      = 4
	FovY       = 15
	AmbColor   = "#b0b0b0"
	LightColor = "#808080"
)

var (
	eye    = aeno.V(0.75, 0.85, 2)
	center = aeno.V(0, 0.06, 0)
	up     = aeno.V(0, 1, 0)
	light  = aeno.V(-1, 3, 1).Normalize()
)

func main() {
	inputPath := flag.String("in", "", "Path to the .ntsm file")
	outputPath := flag.String("out", "render_test.png", "Path to save the output PNG")
	sparkle := flag.Bool("sparkle", false, "Enable and simulate particle emitters")
	flag.Parse()

	if *inputPath == "" {
		fmt.Println("Usage: ntsm-test-render -in <file.ntsm> [-out output.png] [--sparkle]")
		os.Exit(1)
	}

	f, err := os.Open(*inputPath)
	if err != nil {
		log.Fatalf("Failed to open NTSM: %v", err)
	}
	defer f.Close()

	loaded, err := aenoAdapter.LoadObject(f)
	if err != nil {
		log.Fatalf("Failed to decode NTSM: %v", err)
	}

	fmt.Printf("Loaded: %s\n", loaded.Name)
	fmt.Printf("Particles: %d emitters found\n", len(loaded.Emitters))

	var sceneObjects []*aeno.Object
	
	if loaded.Object != nil {
		sceneObjects = append(sceneObjects, loaded.Object)
	}

	if *sparkle && loaded.ParticleSystem != nil {
		fmt.Println("Simulating sparkles...")
		
		viewMatrix := aeno.LookAt(eye, center, up)
		deltaTime := 1.0 / 60.0
		for i := 0; i < 60; i++ {
			loaded.ParticleSystem.Update(deltaTime, aeno.Identity())
		}

		particleObjs := loaded.ParticleSystem.GetObjects(viewMatrix)
		fmt.Printf("Generated %d particle instances for render\n", len(particleObjs))
		sceneObjects = append(sceneObjects, particleObjs...)
	} else if *sparkle {
		fmt.Println("Warning: --sparkle requested but no emitters found in file.")
	}

	fmt.Printf("Rendering scene with %d total objects...\n", len(sceneObjects))
	
	out, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer out.Close()

	err = aeno.GenerateSceneToWriter(
		out,
		sceneObjects,
		eye,
		center,
		up,
		FovY,
		Dimensions,
		Scale,
		light,
		AmbColor,
		LightColor,
		1,
		1000,
		true,
	)

	if err != nil {
		log.Fatalf("Render failed: %v", err)
	}

	absPath, _ := filepath.Abs(*outputPath)
	fmt.Printf("\nSUCCESS: Render saved to %s\n", absPath)
}