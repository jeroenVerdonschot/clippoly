package clippoly

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var (
	targetColor    = color.RGBA{R: 30, G: 80, B: 220, A: 255}
	cropColor      = color.RGBA{R: 20, G: 150, B: 20, A: 255}
	highlightColor = color.RGBA{R: 220, G: 40, B: 40, A: 255}
)

func fillImage(img *image.RGBA, col color.Color) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Set(x, y, col)
		}
	}
}

func drawLineRGBA(img *image.RGBA, x0, y0, x1, y1 int, col color.Color) {
	dx := absInt(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInt(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy

	for {
		setPixel(img, x0, y0, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func setPixel(img *image.RGBA, x, y int, col color.Color) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	img.Set(x, y, col)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sanitizeFilename(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "testcase"
	}
	return b.String()
}

func drawPointRGBA(img *image.RGBA, x, y int, col color.Color) {
	const radius = 2
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			setPixel(img, x+dx, y+dy, col)
		}
	}
}

func saveIntersectPNG(path string, a1, a2, b1, b2 Coord, extras ...Coord) error {
	groups := [][]Coord{{a1, a2}, {b1, b2}}
	if len(extras) > 0 {
		groups = append(groups, extras)
	}
	minX, maxX, minY, maxY, ok := coordBounds(groups...)
	if !ok {
		return nil
	}

	spanX := maxX - minX
	spanY := maxY - minY
	if spanX == 0 {
		spanX = 1
	}
	if spanY == 0 {
		spanY = 1
	}

	maxSpan := spanX
	if spanY > maxSpan {
		maxSpan = spanY
	}

	const maxDim = 256.0
	scale := maxDim / maxSpan
	if scale < 1 {
		scale = 1
	}

	const margin = 12.0
	width := int(math.Ceil(spanX*scale + margin*2))
	height := int(math.Ceil(spanY*scale + margin*2))
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillImage(img, color.RGBA{R: 245, G: 245, B: 245, A: 255})

	project := func(c Coord) (int, int) {
		x := (float64(c[0])-minX)*scale + margin
		y := (maxY-float64(c[1]))*scale + margin
		return int(math.Round(x)), int(math.Round(y))
	}

	ax0, ay0 := project(a1)
	ax1, ay1 := project(a2)
	bx0, by0 := project(b1)
	bx1, by1 := project(b2)
	drawLineRGBA(img, ax0, ay0, ax1, ay1, color.RGBA{R: 30, G: 80, B: 220, A: 255})
	drawLineRGBA(img, bx0, by0, bx1, by1, color.RGBA{R: 220, G: 40, B: 40, A: 255})
	for _, c := range extras {
		x, y := project(c)
		drawPointRGBA(img, x, y, highlightColor)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}
