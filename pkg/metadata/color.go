package metadata

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	_ "golang.org/x/image/webp"
)

// ExtractDominantColor extracts the most vibrant dominant color from the cover art image.
// Returns a hex string like "#7F77DD".
func ExtractDominantColor(imgPath string) string {
	if imgPath == "" {
		return "#7F77DD"
	}

	file, err := os.Open(imgPath)
	if err != nil {
		return "#7F77DD"
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "#7F77DD"
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	stepX := w / 35
	if stepX < 1 {
		stepX = 1
	}
	stepY := h / 35
	if stepY < 1 {
		stepY = 1
	}

	var bestR, bestG, bestB float64
	var maxScore float64
	found := false

	var sumR, sumG, sumB float64
	var count float64

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			c := img.At(x, y)
			r32, g32, b32, _ := c.RGBA()
			r := float64(r32 >> 8)
			g := float64(g32 >> 8)
			b := float64(b32 >> 8)

			hVal, sVal, lVal := rgbToHSL(r, g, b)
			_ = hVal

			// Filter out black/white/gray background pixels
			if lVal > 0.15 && lVal < 0.85 && sVal > 0.18 {
				score := sVal * (1.0 - math.Abs(lVal-0.5))
				if score > maxScore {
					maxScore = score
					bestR, bestG, bestB = r, g, b
					found = true
				}
			}

			sumR += r
			sumG += g
			sumB += b
			count++
		}
	}

	if !found && count > 0 {
		bestR = sumR / count
		bestG = sumG / count
		bestB = sumB / count
	}

	if bestR == 0 && bestG == 0 && bestB == 0 {
		return "#7F77DD"
	}

	return fmt.Sprintf("#%02X%02X%02X", int(bestR), int(bestG), int(bestB))
}

func rgbToHSL(r, g, b float64) (h, s, l float64) {
	r /= 255.0
	g /= 255.0
	b /= 255.0

	maxC := math.Max(r, math.Max(g, b))
	minC := math.Min(r, math.Min(g, b))
	delta := maxC - minC

	l = (maxC + minC) / 2.0

	if delta == 0 {
		h = 0
		s = 0
	} else {
		if l < 0.5 {
			s = delta / (maxC + minC)
		} else {
			s = delta / (2.0 - maxC - minC)
		}

		if maxC == r {
			h = (g - b) / delta
			if g < b {
				h += 6
			}
		} else if maxC == g {
			h = (b-r)/delta + 2
		} else {
			h = (r-g)/delta + 4
		}
		h /= 6
	}

	return h, s, l
}
