package main

import (
	"fmt"
	"log"
	"os"

	aenoAdapter "github.com/netisu/ntsm/adapters/aeno"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: basic_load <ntsm-file>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	loaded, err := aenoAdapter.LoadObject(f)
	if err != nil {
		log.Fatalf("Failed to load NTSM: %v", err)
	}

	fmt.Printf("Successfully loaded %s\n", filePath)
	fmt.Printf("Item name: %s\n", string(loaded.Name[:]))
	fmt.Printf("Contains %d particle emitters\n", len(loaded.Emitters))
	fmt.Printf("GLB size: %d bytes\n", len(loaded.GLBData))

	if loaded.Object != nil && loaded.Object.Mesh != nil {
		fmt.Printf("Mesh has %d lines\n", len(loaded.Object.Mesh.Lines))
		fmt.Printf("Mesh has %d triangles\n", len(loaded.Object.Mesh.Triangles))
	} else {
		fmt.Println("No mesh data found")
	}
}
