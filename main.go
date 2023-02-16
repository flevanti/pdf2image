package main

import (
	"flag"
	"fmt"
	"github.com/gen2brain/go-fitz"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var filename string
var pageFirst int
var pageLast int
var allPages bool
var pagesToProcess int

var wg sync.WaitGroup

const firstPageForAllPages int = 1
const lastPageForAllPages int = 999

//TODO PASS DPI AS PARAMETER
//TODO PASS FORMAT AS PARAMETER (PNG? JPEG?)
//TODO PASS PARAMETER TO PROCESS THINGS SEQUENTIALLY INSTEAD OF CONCURRENCY...
//TODO PASS TARGET DIRECTORY & FILENAME PREFIX AS PARAMETER
//TODO RETURN A LIST OF FILES GENERATED

func main() {

	flag.StringVar(&filename, "filename", "", "Provide a filename")
	flag.IntVar(&pageFirst, "first", firstPageForAllPages, "First page to process")
	flag.IntVar(&pageLast, "last", lastPageForAllPages, "Last page to process")

	flag.Parse()

	if filename == "" {
		fmt.Println("No filename provided")
		os.Exit(1)
	}

	if pageFirst > pageLast {
		fmt.Println("First page to process cannot be greater than the last page!")
		os.Exit(1)
	}

	if pageFirst < firstPageForAllPages {
		fmt.Printf("First page must be greater than %d\n", firstPageForAllPages-1)
		os.Exit(1)
	}

	if pageLast > lastPageForAllPages {
		fmt.Printf("Last page must be less than %d\n", lastPageForAllPages+1)
		os.Exit(1)
	}

	allPages = pageFirst == firstPageForAllPages && pageLast == lastPageForAllPages

	if allPages {
		fmt.Printf("Processing file [%s]\n", filename)

	} else {
		fmt.Printf("Processing file [%s], pages [%d]...[%d]\n", filename, pageFirst, pageLast)
		pagesToProcess = pageLast - pageFirst + 1
	}

	tsStart := time.Now()

	convert()

	fmt.Printf("It took %v\n", time.Since(tsStart))

}
func convert() {

	numpages := 0
	doc, err := fitz.New(filename)
	if err != nil {
		panic(err)
	}
	numpages = doc.NumPage()
	if pageLast > numpages {
		fmt.Printf("Last page requested [%d] exceeds the number of page in the document [%d]\n", pageLast, numpages)
	}
	if allPages {
		pagesToProcess = numpages
	}
	fmt.Printf("File [%s] opened (%d pages)\n", filename, numpages)
	defer doc.Close()

	tmpDir := "output-" + time.Now().Format("20060102150405")
	err = os.Mkdir(tmpDir, fs.ModePerm)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Temp directory is %s\n", tmpDir)

	//pages processed
	c := 0

	// Extract pages as images

	for n := pageFirst; n <= numpages; n++ {
		if !allPages && (n < pageFirst || n > pageLast) {
			continue
		}

		if err != nil {
			panic(err)
		}
		c++
		fmt.Printf("Scanning page %d of %d (exported so far %d/%d)               \r", n, numpages, c, pagesToProcess)
		wg.Add(1)
		//CONCURRENCY HAS BEEN ADDED FOR FUN
		//IT WOULD BE BETTER TO HAVE FLAG TO DECIDE TO USE IT OR NOT
		//AND HAVE MORE VISIBILITY IF CONCURRENCY IS USED OTHERWISE THE SYSTEM WILL JUST WAIT FOR THE PROCESS TO FINISH... BORING...
		go func(pg int) {
			defer wg.Done()
			//please note n-1, pages in the doc start at zero
			img, err := doc.ImagePNG(pg-1, 150)
			if err != nil {
				panic(err)
			}

			f, err := os.Create(filepath.Join(tmpDir, fmt.Sprintf("test%03d.png", pg)))
			if err != nil {
				panic(err)
			}
			//bytes value returned is discarded....
			_, err = f.Write(img)
			if err != nil {
				panic(err)
			}

			f.Close()
		}(n)

		if !allPages && n > pageLast {
			break
		}
	} //end for loop

	fmt.Println()

	fmt.Println("Waiting for the go funcs to complete.... ")
	wg.Wait()
}
