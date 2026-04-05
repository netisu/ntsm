package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/netisu/ntsm"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	HeaderSize       = 192
	TextureEntrySize = 72
)

type NTSMHeader struct {
	Magic          [4]byte
	Version        uint32
	Name           [128]byte
	Flags          uint8
	_              [3]byte
	GLBOffset      uint32
	GLBSize        uint32
	ParticleOffset uint32
	ParticleSize   uint32
	TextureCount   uint32
	TextureOffset  uint32
	ScriptCount    uint32
	ScriptOffset   uint32
	_              [20]byte
}

type TextureTableEntry struct {
	Name   [64]byte
	Size   uint32
	Offset uint32
}

func main() {
	srcDir := flag.String("src", "./uploads", "Source directory containing .obj/.glb files")
	dstDir := flag.String("dst", "./uploads-ntsm", "Destination directory for .ntsm files")
	concurrency := flag.Int("concurrency", 4, "Number of concurrent conversions")
	dryRun := flag.Bool("dry-run", false, "Preview conversions without writing files")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	confirm := flag.Bool("yes", false, "Skip confirmation prompt")
	flag.Parse()

	if _, err := exec.LookPath("obj2gltf"); err != nil {
		log.Fatalf("obj2gltf is not installed. Please install it with: bun install -g obj2gltf")
	}

	files, err := findSourceFiles(*srcDir)
	if err != nil {
		log.Fatalf("Failed to scan source directory: %v", err)
	}

	if !*confirm {
		fmt.Printf("Found %d assets. Proceed? [y/N] ", len(files))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
			os.Exit(0)
		}
	}

	success, failed := processFiles(files, *srcDir, *dstDir, *concurrency, *dryRun, *verbose)
	fmt.Printf("\nCompleted: %d success, %d failed\n", success, failed)
}

func findSourceFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && (strings.HasSuffix(path, ".obj") || strings.HasSuffix(path, ".glb")) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func processFiles(files []string, srcDir, dstDir string, concurrency int, dryRun, verbose bool) (int, int) {
	var wg sync.WaitGroup
	var success, failed int
	var mu sync.Mutex

	tasks := make(chan string, len(files))
	for _, f := range files {
		tasks <- f
	}
	close(tasks)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for src := range tasks {
				rel, _ := filepath.Rel(srcDir, src)
				dst := filepath.Join(dstDir, strings.TrimSuffix(rel, filepath.Ext(rel))+".ntsm")

				if err := convertToNTSM(src, dst, dryRun, verbose); err != nil {
					mu.Lock()
					failed++
					mu.Unlock()
					fmt.Printf("Error %s: %v\n", src, err)
				} else {
					mu.Lock()
					success++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return success, failed
}

func convertToNTSM(srcPath, dstPath string, dryRun bool, verbose bool) error {
	var glbData []byte
	var err error

	if strings.HasSuffix(srcPath, ".obj") {
		tempGLB := srcPath + ".temp.glb"
		cmd := exec.Command("obj2gltf", "-b", "-i", srcPath, "-o", tempGLB)
		if err = cmd.Run(); err != nil {
			return err
		}
		glbData, _ = os.ReadFile(tempGLB)
		os.Remove(tempGLB)
	} else {
		glbData, _ = os.ReadFile(srcPath)
	}

	var textures [][]byte
	var entries []TextureTableEntry

	modelTexPath := strings.TrimSuffix(srcPath, filepath.Ext(srcPath)) + ".png"
	if data, err := os.ReadFile(modelTexPath); err == nil {
		textures = append(textures, data)
		var entry TextureTableEntry
		copy(entry.Name[:], []byte(filepath.Base(modelTexPath)))
		entries = append(entries, entry)
	}

	sparklePath := filepath.Join(filepath.Dir(srcPath), "sparkle.png")
	sparkleIndex := -1
	if data, err := os.ReadFile(sparklePath); err == nil {
		sparkleIndex = len(textures)
		textures = append(textures, data)
		var entry TextureTableEntry
		copy(entry.Name[:], []byte("sparkle.png"))
		entries = append(entries, entry)
	}

	var emitters []ntsm.ParticleEmitter
	if sparkleIndex != -1 {
		emitters = append(emitters, ntsm.ParticleEmitter{
			Position:         [3]float32{0, 1.5, 0},
			Direction:        [3]float32{0, 1, 0},
			SpreadAngle:      6.28,
			EmissionRate:     30,
			ParticleLifetime: 1.3,
			StartSize:        0.37,
			EndSize:          0.37,
			StartColor:       [4]float32{1, 1, 0.8, 1}, // Pale yellow
			EndColor:         [4]float32{1, 0.5, 0, 0}, // Fading orange
			TextureIndex:     int32(sparkleIndex),
			MaxParticles:     100,
			SpawnCount:       1,
			BlendMode:        0,
			Loop:             0,
			VelocityMin:      [3]float32{0, 0, 0},
			VelocityMax:      [3]float32{0, 0, 0},
		})
	}

	h := NTSMHeader{
		Magic:     [4]byte{'N', 'T', 'S', 'M'},
		Version:   1,
		GLBOffset: HeaderSize,
		GLBSize:   uint32(len(glbData)),
	}

	h.ParticleOffset = h.GLBOffset + h.GLBSize
	emitterSize := 0
	if len(emitters) > 0 {
		emitterSize = binary.Size(emitters[0])
	}
	h.ParticleSize = uint32(len(emitters) * emitterSize)

	h.ScriptCount = 0
	h.ScriptOffset = h.ParticleOffset + h.ParticleSize

	h.TextureCount = uint32(len(textures))
	h.TextureOffset = h.ScriptOffset

	tableSize := h.TextureCount * TextureEntrySize
	currentDataOffset := h.TextureOffset + tableSize

	for i := range entries {
		entries[i].Size = uint32(len(textures[i]))
		entries[i].Offset = currentDataOffset
		currentDataOffset += entries[i].Size
	}

	out, _ := os.Create(dstPath)
	defer out.Close()

	binary.Write(out, binary.LittleEndian, h)
	out.Write(glbData)

	for _, e := range emitters {
		binary.Write(out, binary.LittleEndian, e)
	}

	for _, entry := range entries {
		binary.Write(out, binary.LittleEndian, entry)
	}

	for _, texData := range textures {
		out.Write(texData)
	}

	return nil
}
