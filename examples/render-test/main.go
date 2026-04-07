package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time" // Added for duration tracking

	"aeno"
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
	
	var sceneObjects []*aeno.Object
	
	if loaded.Object != nil {
		sceneObjects = append(sceneObjects, loaded.Object)
	}

	if *sparkle && loaded.ParticleSystem != nil {
		fmt.Println("Simulating sparkles (3s duration)...")
		
		viewMatrix := aeno.LookAt(eye, center, up)
		deltaTime := 1.0 / 60.0
		
		modelMat := aeno.Identity()
		if loaded.Object != nil {
			modelMat = loaded.Object.Matrix
		}
		for i := 0; i < 180; i++ { 
			loaded.ParticleSystem.Update(deltaTime, modelMat)
		}

		particleObjs := loaded.ParticleSystem.GetObjects(viewMatrix)
		fmt.Printf("Generated %d particle instances for render\n", len(particleObjs))
		sceneObjects = append(sceneObjects, particleObjs...)
	}

	fmt.Printf("Rendering scene with %d total objects...\n", len(sceneObjects))
	
	out, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer out.Close()

	startTime := time.Now()

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

	renderDuration := time.Since(startTime)

	if err != nil {
		log.Fatalf("Render failed: %v", err)
	}

	absPath, _ := filepath.Abs(*outputPath)
	fmt.Printf("\nSUCCESS: Render saved to %s\n", absPath)
	fmt.Printf("Render Time: %v\n", renderDuration)
}