package metadata

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// RenderCoverArt renders the image at coverPath as ANSI TrueColor block art.
// It automatically crops non-square 16:9 images (like 1280x720 YouTube thumbnails)
// to a clean 1:1 central square, eliminating side letterbox borders.
func RenderCoverArt(coverPath string, width, height int) string {
	if coverPath == "" {
		return renderPlaceholder(width, height)
	}

	file, err := os.Open(coverPath)
	if err != nil {
		return renderPlaceholder(width, height)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return renderPlaceholder(width, height)
	}

	// Auto-crop non-square images (like YouTube 1280x720 thumbnails) to central 1:1 square
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	cropBounds := srcBounds
	if srcW > srcH {
		offsetX := (srcW - srcH) / 2
		cropBounds = image.Rect(srcBounds.Min.X+offsetX, srcBounds.Min.Y, srcBounds.Min.X+offsetX+srcH, srcBounds.Max.Y)
	} else if srcH > srcW {
		offsetY := (srcH - srcW) / 2
		cropBounds = image.Rect(srcBounds.Min.X, srcBounds.Min.Y+offsetY, srcBounds.Max.X, srcBounds.Min.Y+offsetY+srcW)
	}

	// Calculate target pixel dimensions: height character cells = height*2 pixels
	pixelWidth := width
	pixelHeight := height * 2

	dst := image.NewRGBA(image.Rect(0, 0, pixelWidth, pixelHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, cropBounds, draw.Over, nil)

	var lines []string
	for y := 0; y < pixelHeight; y += 2 {
		var lineBuilder strings.Builder
		for x := 0; x < pixelWidth; x++ {
			topColor := dst.At(x, y)
			botColor := dst.At(x, y+1)

			r1, g1, b1, _ := colorToRGB(topColor)
			r2, g2, b2, _ := colorToRGB(botColor)

			// Half-block character '▀' with foreground (top) and background (bottom)
			cell := fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
				r1, g1, b1, r2, g2, b2)
			lineBuilder.WriteString(cell)
		}
		lines = append(lines, lineBuilder.String())
	}

	return strings.Join(lines, "\n")
}

func colorToRGB(c color.Color) (r, g, b, a uint8) {
	r32, g32, b32, a32 := c.RGBA()
	return uint8(r32 >> 8), uint8(g32 >> 8), uint8(b32 >> 8), uint8(a32 >> 8)
}

func renderPlaceholder(width, height int) string {
	var lines []string
	for y := 0; y < height; y++ {
		var line string
		if y == height/2 {
			line = fmt.Sprintf("│%-*s│", width-2, "   ♫  ShellBeat")
		} else if y == 0 {
			line = "┌" + strings.Repeat("─", width-2) + "┐"
		} else if y == height-1 {
			line = "└" + strings.Repeat("─", width-2) + "┘"
		} else {
			line = "│" + strings.Repeat(" ", width-2) + "│"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
