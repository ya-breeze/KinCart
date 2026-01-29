package flyers

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder for image.Decode
	"image/png"
	"os"
	"path/filepath"

	"kincart/internal/utils"

	"github.com/google/uuid"
)

// CropItem crops an image based on normalized bounding box [ymin, xmin, ymax, xmax] (0-1000)
// and saves it to a local file. Returns the local path.
func CropItem(imageData []byte, box []float64, outputDir string, itemName string) (string, error) {
	if len(box) != 4 {
		return "", fmt.Errorf("invalid bounding box: %v", box)
	}

	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Add 5% padding to the normalized coordinates
	padding := 5.0
	yminNorm := box[0] - padding
	xminNorm := box[1] - padding
	ymaxNorm := box[2] + padding
	xmaxNorm := box[3] + padding

	// Clamp normalized values
	if yminNorm < 0 {
		yminNorm = 0
	}
	if xminNorm < 0 {
		xminNorm = 0
	}
	if ymaxNorm > 1000 {
		ymaxNorm = 1000
	}
	if xmaxNorm > 1000 {
		xmaxNorm = 1000
	}

	// Normalize coordinates (0-1000) to actual pixels
	ymin := int(yminNorm * float64(height) / 1000.0)
	xmin := int(xminNorm * float64(width) / 1000.0)
	ymax := int(ymaxNorm * float64(height) / 1000.0)
	xmax := int(xmaxNorm * float64(width) / 1000.0)

	// Clamp to bounds
	if xmin < 0 {
		xmin = 0
	}
	if ymin < 0 {
		ymin = 0
	}
	if xmax > width {
		xmax = width
	}
	if ymax > height {
		ymax = height
	}

	rect := image.Rect(0, 0, xmax-xmin, ymax-ymin)
	cropped := image.NewRGBA(rect)
	draw.Draw(cropped, rect, img, image.Point{xmin, ymin}, draw.Src)

	filename := fmt.Sprintf("%s.png", uuid.New().String())
	shardDir := utils.GetShardDir(outputDir, filename)

	if err = os.MkdirAll(shardDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(shardDir, filename)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, cropped); err != nil {
		return "", fmt.Errorf("failed to encode png: %w", err)
	}

	return outputPath, nil
}
