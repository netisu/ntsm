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

	fileInfo, err := f.Stat()
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	fmt.Printf("File size: %d bytes\n", fileInfo.Size())

	header := make([]byte, 4)
	f.ReadAt(header, 0)
	fmt.Printf("Header magic: %q (should be \"NTSM\")\n", string(header))

	glbCheck := make([]byte, 4)
	f.ReadAt(glbCheck, 192)
	fmt.Printf("GLB section start: %q (should be \"glTF\")\n", string(glbCheck))

	f.Seek(0, 0)

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
	
	fmt.Printf("Contains %d particle emitters\n", len(loaded.Emitters))
}
