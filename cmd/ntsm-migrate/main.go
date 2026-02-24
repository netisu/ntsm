package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

	// Check if obj2gltf is installed
	if _, err := exec.LookPath("obj2gltf"); err != nil {
		log.Fatalf("obj2gltf is not installed. Please install it with: bun install -g obj2gltf")
	}

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

				if err := convertToNTSM(file, dstPath, dryRun, verbose); err != nil {
					counter.Lock()
					counter.failed++
					counter.Unlock()
					if verbose {
						fmt.Printf("Failed: %v\n", err)
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

func convertToNTSM(srcPath, dstPath string, dryRun bool, verbose bool) error {
	var glbData []byte
	var err error

	if strings.HasSuffix(srcPath, ".obj") {
		if verbose {
			fmt.Printf("[worker] Converting .obj to GLB: %s\n", srcPath)
		}

		tempGLBPath := srcPath + ".temp.glb"

		cmd := exec.Command("obj2gltf", "-i", srcPath, "-o", tempGLBPath)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		if err = cmd.Run(); err != nil {
			return fmt.Errorf("[worker] obj2gltf conversion failed: %w", err)
		}

		glbData, err = os.ReadFile(tempGLBPath)
		if err != nil {
			return fmt.Errorf("[worker] failed to read converted GLB: %w", err)
		}

		if !dryRun && !verbose {
			os.Remove(tempGLBPath)
		}
	} else {
		glbData, err = os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("[worker] read failed: %w", err)
		}
	}

	header := createHeader(srcPath, glbData)

	if dryRun {
		return nil
	}

	if err = os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("[worker] mkdir failed: %w", err)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("[worker] create failed: %w", err)
	}
	defer out.Close()

	if err = binary.Write(out, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("[worker] header write failed: %w", err)
	}

	padding := make([]byte, HeaderSize-192)
	if _, err = out.Write(padding); err != nil {
		return fmt.Errorf("[worker] padding write failed: %w", err)
	}

	if _, err = out.Write(glbData); err != nil {
		return fmt.Errorf("[worker] glb write failed: %w", err)
	}

	return nil
}

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
