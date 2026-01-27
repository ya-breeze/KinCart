package flyers

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/gen2brain/go-fitz"
)

// SplitPDF converts a PDF file into a set of PNG images, one for each page.
func SplitPDF(pdfData []byte, outputDir string) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to open pdf: %w", err)
	}
	defer doc.Close()

	var files []string
	for i := 0; i < doc.NumPage(); i++ {
		img, err := doc.Image(i)
		if err != nil {
			return nil, fmt.Errorf("failed to render page %d: %w", i, err)
		}

		outPath := filepath.Join(outputDir, fmt.Sprintf("page_%d.png", i+1))
		f, err := os.Create(outPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}

		err = png.Encode(f, img)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to encode png for page %d: %w", i, err)
		}

		files = append(files, outPath)
	}

	return files, nil
}
