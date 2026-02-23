package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/netisu/aeno"
)

const (
	HeaderSize = 192 // Fixed header size in bytes
)

// NTSMHeader represents the binary header of the NTSM file format
type NTSMHeader struct {
	Magic          [4]byte // "NTSM"
	Version        uint32  // Format version (1)
	Name           [128]byte
	Flags          uint8   // Bitfield: bit 0 = has_particles
	_              [3]byte // Padding
	GLBOffset      uint32  // Offset to GLB data
	GLBSize        uint32  // Size of GLB data
	ParticleOffset uint32  // Offset to particle data
	ParticleSize   uint32  // Size of particle data
	TextureCount   uint32  // Number of embedded textures
	TextureOffset  uint32  // Offset to texture table
}

// ParticleEmitter represents a single particle system configuration
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

func main() {
	srcDir := flag.String("src", "./uploads", "Source directory containing .obj/.glb files")
	dstDir := flag.String("dst", "./uploads-ntsm", "Destination directory for .ntsm files")
	concurrency := flag.Int("concurrency", 4, "Number of concurrent conversions")
	dryRun := flag.Bool("dry-run", false, "Preview conversions without writing files")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	confirm := flag.Bool("yes", false, "Skip confirmation prompt")
	flag.Parse()

	srcInfo, err := os.Stat(*srcDir)
	if err != nil || !srcInfo.IsDir() {
		log.Fatalf("Source directory does not exist: %s", *srcDir)
	}

	if err := os.MkdirAll(*dstDir, 0755); err != nil {
		log.Fatalf("Failed to create destination directory: %v", err)
	}

	files, err := findSourceFiles(*srcDir)
	if err != nil {
		log.Fatalf("Failed to scan source directory: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("No .obj or .glb files found in %s", *srcDir)
	}

	fmt.Printf("Found %d assets to convert:\n", len(files))
	for i, f := range files {
		if i < 10 || i >= len(files)-5 {
			fmt.Printf("  %s\n", f)
		} else if i == 10 {
			fmt.Printf("  ...\n")
		}
	}
	fmt.Printf("\nSource: %s\n", *srcDir)
	fmt.Printf("Destination: %s\n", *dstDir)
	fmt.Printf("Concurrency: %d workers\n", *concurrency)
	if *dryRun {
		fmt.Println("Mode: DRY RUN (no files will be written)")
	}

	if !*confirm {
		fmt.Print("\nProceed with migration? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("Migration aborted.")
			os.Exit(0)
		}
	}

	start := time.Now()
	success, failed := processFiles(files, *srcDir, *dstDir, *concurrency, *dryRun, *verbose)

	duration := time.Since(start).Truncate(time.Millisecond)
	fmt.Printf("\nMigration completed in %v\n", duration)
	fmt.Printf("✓ Successfully converted: %d\n", success)
	fmt.Printf("✗ Failed: %d\n", failed)

	if failed > 0 && !*dryRun {
		fmt.Println("\nTip: Check logs for details on failed conversions.")
		fmt.Println("You can retry individual files with: ntsm-migrate -src <file> -dst <file.ntsm>")
	}
}

func findSourceFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".obj") || strings.HasSuffix(path, ".glb")) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// processFiles converts multiple files with concurrency control
func processFiles(files []string, srcDir, dstDir string, concurrency int, dryRun, verbose bool) (int, int) {
	var (
		wg      sync.WaitGroup
		counter struct {
			sync.Mutex
			success, failed int
		}
	)

	tasks := make(chan string, len(files))
	for _, file := range files {
		tasks <- file
	}
	close(tasks)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range tasks {
				relPath, err := filepath.Rel(srcDir, file)
				if err != nil {
					relPath = file
				}
				dstPath := filepath.Join(dstDir, strings.TrimSuffix(relPath, filepath.Ext(relPath))+".ntsm")

				if verbose {
					fmt.Printf("[worker] Converting %s → %s\n", relPath, dstPath)
				}

				if err := convertToNTSM(file, dstPath, dryRun); err != nil {
					counter.Lock()
					counter.failed++
					counter.Unlock()
					if verbose {
						fmt.Printf("Failed: %s\n", err)
					}
				} else {
					counter.Lock()
					counter.success++
					counter.Unlock()
					if verbose {
						fmt.Printf("Converted: %s\n", relPath)
					}
				}
			}
		}()
	}

	wg.Wait()
	return counter.success, counter.failed
}

func convertToNTSM(srcPath, dstPath string, dryRun bool) error {
	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("[worker] read failed: %w", err)
	}

	var glbData []byte
	if strings.HasSuffix(srcPath, ".obj") {
		if verbose {
			fmt.Printf("[worker] Converting .obj to GLB: %s\n", srcPath)
		}
		mesh, err := aeno.LoadOBJFromReader(bytes.NewReader(srcData))
		if err != nil {
			return fmt.Errorf("[worker] obj parse failed: %w", err)
		}

		var buf bytes.Buffer
		if err := saveGLBFromMesh(&buf, mesh); err != nil {
			return fmt.Errorf("[worker] glb export failed: %w", err)
		}
		glbData = buf.Bytes()
	} else {
		glbData = srcData
	}

	header := createHeader(srcPath, glbData)

	if dryRun {
		return nil // Skip writing
	}

	// Write NTSM file
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("[worker] mkdir failed: %w", err)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("[worker] create failed: %w", err)
	}
	defer out.Close()

	if err := binary.Write(out, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("[worker] header write failed: %w", err)
	}

	padding := make([]byte, HeaderSize-192)
	if _, err := out.Write(padding); err != nil {
		return fmt.Errorf("[worker] padding write failed: %w", err)
	}

	if _, err := out.Write(glbData); err != nil {
		return fmt.Errorf("[worker] glb write failed: %w", err)
	}

	return nil
}

// createHeader builds the NTSM header for a given asset
func createHeader(srcPath string, glbData []byte) NTSMHeader {
	base := filepath.Base(srcPath)
	itemID := strings.TrimSuffix(base, filepath.Ext(base))

	header := NTSMHeader{
		Version:        1,
		GLBOffset:      HeaderSize,
		GLBSize:        uint32(len(glbData)),
		ParticleOffset: HeaderSize + uint32(len(glbData)),
		ParticleSize:   0,
		TextureCount:   0,
		TextureOffset:  0,
		Flags:          0,
	}

	copy(header.Magic[:], []byte("NTSM"))

	name := []byte(itemID)
	if len(name) > 127 {
		name = name[:127]
	}
	copy(header.Name[:], name)

	return header
}

func saveGLBFromMesh(w io.Writer, mesh *aeno.Mesh) error {
	obj := &aeno.Object{
		Mesh:   mesh,
		Matrix: aeno.Identity(),
	}

	var buf bytes.Buffer
	if err := aeno.GenerateSceneToWriter(&buf, []*aeno.Object{obj},
		aeno.V(0, 0, 2), aeno.V(0, 0, 0), aeno.V(0, 1, 0),
		45, 512, 1, aeno.V(-1, 1, 1), "#ffffff", "#ffffff", 0.1, 100, true); err != nil {
		return err
	}

	_, err := w.Write(buf.Bytes())
	return err
}
