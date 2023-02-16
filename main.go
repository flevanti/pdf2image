package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gen2brain/go-fitz"
)

func main() {
	PrintMemUsage()
	convert()
	PrintMemUsage()
	runtime.GC()
	PrintMemUsage()
}
func convert() {

	filename := "test2.pdf"
	numpages := 0
	doc, err := fitz.New(filename)
	if err != nil {
		panic(err)
	}
	numpages = doc.NumPage()
	log.Printf("File [%s] opened (%d pages)", filename, numpages)
	defer doc.Close()

	tmpDir, err := os.MkdirTemp(".", "output-")
	if err != nil {
		panic(err)
	}

	log.Printf("Temp directory is %s", tmpDir)

	// Extract pages as images
	for n := 0; n < doc.NumPage(); n++ {
		log.Printf("Converting page %d of %d", n, numpages)
		img, err := doc.ImagePNG(n, 300)
		if err != nil {
			panic(err)
		}

		f, err := os.Create(filepath.Join(tmpDir, fmt.Sprintf("test%03d.png", n)))
		if err != nil {
			panic(err)
		}
		//bytes value returned is discarded....
		_, err = f.Write(img)
		if err != nil {
			panic(err)
		}

		f.Close()
	}

}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
