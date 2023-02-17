package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gen2brain/go-fitz"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type MetadataGeneratedT struct {
	Wd           string           `json:"wd"`
	Folder       string           `json:"folder"`
	PdfFilename  string           `json:"pdffilename"`
	PagesInRange int              `json:"pagesinrange"`
	PageFirst    int              `json:"pagefirst"`
	PageLast     int              `json:"pagelast"`
	PagesInfile  int              `json:"pagesinfile"`
	AllPages     bool             `json:"allpages"`
	Dpi          int              `json:"dpi"`
	Started      int64            `json:"started"`
	Completed    int64            `json:"completed"`
	SecondsSpent int64            `json:"secondsspent"`
	TotalBytes   int              `json:"totalbytes"`
	Filenames    []FileGeneratedT `json:"filenames"`
}

type FileGeneratedT struct {
	Name  string `json:"name"`
	Bytes int    `json:"bytes"`
}

var doc *fitz.Document
var Mdg MetadataGeneratedT

const firstPageForAllPages int = 1
const lastPageForAllPages int = 999
const dpiMin = 50
const dpiDefault = 150
const dpiMax = 1000

//TODO PASS DPI AS PARAMETER
//TODO PASS FORMAT AS PARAMETER (PNG? JPEG?)
//TODO PASS TARGET DIRECTORY & FILENAME PREFIX AS PARAMETER

func main() {
	retrieveFlags()
	getwd()
	getTargetFolder()
	if Mdg.PdfFilename == "" {
		exitBad("No filename provided")
	}
	openPdfFile()
	defer func(doc *fitz.Document) {
		err := doc.Close()
		exitOnError(err, fmt.Sprintf("Error closing the pdf file, not that it matters at this point!\n"))
	}(doc)

	Mdg.PagesInfile = doc.NumPage()

	//determine if the user wants all pages or not
	Mdg.AllPages = Mdg.PageFirst == firstPageForAllPages && Mdg.PageLast == lastPageForAllPages
	if Mdg.AllPages {
		//replace the last page wanted with the actual number of pages in the document
		Mdg.PageLast = Mdg.PagesInfile
	}

	//now that we have the real first<->last page calculate the range
	Mdg.PagesInRange = Mdg.PageLast - Mdg.PageFirst + 1

	//Metadata variable populated
	//perform some preflight checks
	preFlightChecks()

	//output some information for the process...
	fmt.Printf("File [%s] opened (%d pages total)\n", Mdg.PdfFilename, Mdg.PagesInfile)
	fmt.Printf("Pages range is [%d...%d] - %d pages requested",
		Mdg.PageFirst, Mdg.PageLast, Mdg.PagesInRange)
	fmt.Printf("Target folder is [%s]\n", Mdg.Folder)

	Mdg.Started = time.Now().Unix()
	convert()
	Mdg.Completed = time.Now().Unix()
	Mdg.SecondsSpent = Mdg.Completed - Mdg.Started
	fmt.Printf("It took %d seconds\n", Mdg.SecondsSpent)
	fmt.Printf("Storage used %.2fMB\n", float64(Mdg.TotalBytes)/1024/1024)

	//don't output the json to avoid noise... but it's here!
	//outputMetadata()
}

func convert() {
	var err error

	err = os.Mkdir(Mdg.Folder, fs.ModePerm)
	exitOnError(err, fmt.Sprintf("Error while creating target Folder [%s]\n", Mdg.Folder))

	// Extract pages as images
	//loop only the number of times required to get the number of pages we need
	for n := 0; n < Mdg.PagesInRange; n++ {
		fmt.Printf("Scanning page %d of %d (exported so far %d/%d)  \r",
			Mdg.PageFirst+n, Mdg.PagesInfile, n+1, Mdg.PagesInRange)
		extractImage(Mdg.PageFirst + n)
	} //end for loop

	fmt.Println()

}

func extractImage(pg int) {
	var err error
	var bytes int
	//please note pg-1, pages in the doc start at zero
	img, err := doc.ImagePNG(pg-1, 150)
	exitOnError(err, fmt.Sprintf("Error converting page [%d] to image\n", pg))

	targetFilename := fmt.Sprintf("image-%04d.png", pg)
	f, err := os.Create(filepath.Join(Mdg.Folder, targetFilename))
	exitOnError(err, fmt.Sprintf("Error creating target filename [%s]\n", targetFilename))

	bytes, err = f.Write(img)
	exitOnError(err, fmt.Sprintf("Error closing target filename [%s]\n", targetFilename))

	err = f.Close()
	exitOnError(err, fmt.Sprintf("Error closing target filename [%s]\n", targetFilename))

	//store information in the metadata variable
	Mdg.Filenames = append(Mdg.Filenames, FileGeneratedT{Name: targetFilename, Bytes: bytes})
	Mdg.TotalBytes += bytes
}

func exitOnError(err error, m string) {
	if err != nil {
		exitBad(fmt.Sprintf("%s\n%s\n\n", m, err.Error()))
	}
}

func exitBad(m string) {
	fmt.Println("🔴 " + m)
	os.Exit(1)
}

func retrieveFlags() {
	flag.StringVar(&Mdg.PdfFilename, "filename", "", "Provide a filename")
	flag.IntVar(&Mdg.PageFirst, "first", firstPageForAllPages, "First page to process")
	flag.IntVar(&Mdg.PageLast, "last", lastPageForAllPages, "Last page to process")
	flag.IntVar(&Mdg.Dpi, "dpi", dpiDefault, "Dpi to use for image generation")

	flag.Parse()
}

func preFlightChecks() {

	//check if page range goes beyond the actual pages in the file
	if Mdg.PageLast > Mdg.PagesInfile {
		exitBad(fmt.Sprintf("Pages range [%d...%d] exceeds pages in document [%d]",
			Mdg.PageFirst, Mdg.PageLast, Mdg.PagesInfile))
	}

	if Mdg.PageFirst > Mdg.PageLast {
		exitBad("First page to process cannot be greater than the last page!")
	}

	if Mdg.PageFirst < firstPageForAllPages {
		exitBad(fmt.Sprintf("First page must be greater than %d\n", firstPageForAllPages-1))
	}

	if Mdg.PageLast > lastPageForAllPages {
		exitBad(fmt.Sprintf("Last page must be less than %d\n", lastPageForAllPages+1))
	}

	if Mdg.Dpi < dpiMin || Mdg.Dpi > dpiMax {
		exitBad(fmt.Sprintf("Dpi value [%d] is outside range [%d - %d]  \n", Mdg.Dpi, dpiMin, dpiMax))
	}

}

func getwd() {
	var err error
	Mdg.Wd, err = os.Getwd()
	exitOnError(err, fmt.Sprintf("Unable to retrieve current working directory\n"))

}

func getTargetFolder() {
	Mdg.Folder = "output-" + time.Now().Format("20060102150405")
}

func outputMetadata() {
	//	b, err := json.MarshalIndent(Mdg, "", "    ")
	b, err := json.Marshal(Mdg)

	exitOnError(err, fmt.Sprintf("Error while converting metadata struct to json"))
	fmt.Println(string(b))

}
func openPdfFile() {
	var err error
	doc, err = fitz.New(Mdg.PdfFilename)
	exitOnError(err, fmt.Sprintf("Unable to open pdf file [%s]", Mdg.PdfFilename))

}
