package clippoly

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func saveTriangleCropPNG(path string, crop Polygon, input Polygon, result Polygons) error {
	cropCoords := trimCoords(crop[:])
	inputCoords := trimCoords(input[:])

	all := [][]Coord{cropCoords, inputCoords}
	for _, tri := range result {
		all = append(all, trimCoords(tri[:]))
	}

	minX, maxX, minY, maxY, ok := coordBounds(all...)
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

	const maxDim = 512.0
	scale := maxDim / maxSpan
	if scale < 1 {
		scale = 1
	}

	const margin = 20.0
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

	drawLoop(img, cropCoords, project, color.RGBA{R: 30, G: 80, B: 220, A: 255}) // blue rectangle
	drawLoop(img, inputCoords, project, color.RGBA{R: 0, G: 0, B: 0, A: 255})    // black input
	for _, tri := range result {
		drawLoop(img, trimCoords(tri[:]), project, color.RGBA{R: 220, G: 40, B: 40, A: 255}) // red results
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

func coordBounds(groups ...[]Coord) (minX, maxX, minY, maxY float64, ok bool) {
	minX, minY = math.MaxFloat64, math.MaxFloat64
	maxX, maxY = -math.MaxFloat64, -math.MaxFloat64

	for _, coords := range groups {
		for _, c := range coords {
			x := float64(c[0])
			y := float64(c[1])

			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
			ok = true
		}
	}

	return
}

func trimCoords(coords []Coord) []Coord {
	if len(coords) <= 1 {
		return coords
	}
	if coords[0] == coords[len(coords)-1] {
		return coords[:len(coords)-1]
	}
	return coords
}

func drawLoop(img *image.RGBA, coords []Coord, project func(Coord) (int, int), col color.Color) {
	if len(coords) == 0 {
		return
	}
	for i := range coords {
		next := (i + 1) % len(coords)
		x0, y0 := project(coords[i])
		x1, y1 := project(coords[next])
		drawLineRGBA(img, x0, y0, x1, y1, col)
	}
}
