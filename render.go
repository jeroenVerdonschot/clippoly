package clippoly

import (
	"image"
	"image/color"
	"strings"
)

var (
	targetColor    = color.RGBA{R: 30, G: 80, B: 220, A: 255}
	cropColor      = color.RGBA{R: 20, G: 150, B: 20, A: 255}
	highlightColor = color.RGBA{R: 220, G: 40, B: 40, A: 255}
)

// func saveScenePNG(name string, target, crop, result polygon, triangulations ...[]polygon) error {
// 	polys := []polygon{}
// 	if target.len() > 0 {
// 		polys = append(polys, target)
// 	}
// 	if crop.len() > 0 {
// 		polys = append(polys, crop)
// 	}
// 	if result.len() > 0 {
// 		polys = append(polys, result)
// 	}
// 	for _, tris := range triangulations {
// 		for _, tri := range tris {
// 			polys = append(polys, tri)
// 		}
// 	}

// 	if len(polys) == 0 {
// 		return nil
// 	}

// 	minX, maxX, minY, maxY, ok := polygonBounds(polys...)
// 	if !ok {
// 		return nil
// 	}

// 	spanX := maxX - minX
// 	spanY := maxY - minY
// 	if spanX == 0 {
// 		spanX = 1
// 	}
// 	if spanY == 0 {
// 		spanY = 1
// 	}
// 	maxSpan := spanX
// 	if spanY > maxSpan {
// 		maxSpan = spanY
// 	}
// 	const maxDim = 512.0
// 	scale := maxDim / maxSpan
// 	if scale < 1 {
// 		scale = 1
// 	}
// 	const margin = 20.0
// 	width := int(math.Ceil(spanX*scale)) + int(margin*2)
// 	height := int(math.Ceil(spanY*scale)) + int(margin*2)
// 	if width < 1 {
// 		width = 1
// 	}
// 	if height < 1 {
// 		height = 1
// 	}

// 	img := image.NewRGBA(image.Rect(0, 0, width, height))
// 	fillImage(img, color.RGBA{R: 245, G: 245, B: 245, A: 255})

// 	project := func(coord [2]float32) (int, int) {
// 		x := (float64(coord[0])-minX)*scale + margin
// 		y := (maxY-float64(coord[1]))*scale + margin
// 		return int(math.Round(x)), int(math.Round(y))
// 	}

// 	type polyEntry struct {
// 		coords polygon
// 		edge   color.RGBA
// 		vertex color.RGBA
// 	}

// 	entries := []polyEntry{
// 		{coords: target, edge: targetColor, vertex: targetColor},
// 		{coords: crop, edge: cropColor, vertex: cropColor},
// 		{coords: result, edge: highlightColor, vertex: highlightColor},
// 	}

// 	for _, entry := range entries {
// 		if len(entry.coords) == 0 {
// 			continue
// 		}
// 		drawPolygonCoords(img, entry.coords, project, entry.edge)
// 		drawVertices(img, entry.coords, project, 3, entry.vertex)
// 	}
// 	for _, tris := range triangulations {
// 		for _, tri := range tris {
// 			triangleCoords := polygon{tri[0], tri[1], tri[2]}
// 			drawPolygonCoords(img, triangleCoords, project, highlightColor)
// 			drawVertices(img, triangleCoords, project, 2, highlightColor)
// 		}
// 	}

// 	dir := filepath.Dir(name)
// 	base := filepath.Base(name)
// 	ext := filepath.Ext(base)
// 	if ext == "" {
// 		ext = ".png"
// 	}
// 	nameOnly := base[:len(base)-len(ext)]
// 	safeBase := sanitizeFilename(nameOnly)
// 	if safeBase == "" {
// 		safeBase = "image"
// 	}
// 	if dir == "" || dir == "." {
// 		dir = "test_output"
// 	}
// 	filename := filepath.Join(dir, safeBase+ext)

// 	if err := os.MkdirAll(dir, 0o755); err != nil {
// 		return err
// 	}

// 	file, err := os.Create(filename)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	return png.Encode(file, img)
// }

// func polygonBounds(polys ...polygon) (minX, maxX, minY, maxY float64, ok bool) {
// 	minX, maxX = math.MaxFloat64, -math.MaxFloat64
// 	minY, maxY = math.MaxFloat64, -math.MaxFloat64

// 	for _, poly := range polys {
// 		for _, coord := range poly {
// 			x := float64(coord[0])
// 			y := float64(coord[1])
// 			if x < minX {
// 				minX = x
// 			}
// 			if x > maxX {
// 				maxX = x
// 			}
// 			if y < minY {
// 				minY = y
// 			}
// 			if y > maxY {
// 				maxY = y
// 			}
// 		}
// 	}

// 	if minX == math.MaxFloat64 {
// 		return 0, 0, 0, 0, false
// 	}
// 	return minX, maxX, minY, maxY, true
// }

// func drawPolygonCoords(img *image.RGBA, coords polygon, project func([2]float32) (int, int), edge color.RGBA) {
// 	if len(coords) == 0 {
// 		return
// 	}
// 	for i := range coords {
// 		next := (i + 1) % len(coords)
// 		x0, y0 := project(coords[i])
// 		x1, y1 := project(coords[next])
// 		drawLineRGBA(img, x0, y0, x1, y1, edge)
// 	}
// }

// func drawVertices(img *image.RGBA, coords polygon, project func([2]float32) (int, int), half int, col color.RGBA) {
// 	for _, coord := range coords {
// 		x, y := project(coord)
// 		drawSquare(img, x, y, half, col)
// 	}
// }

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

func drawSquare(img *image.RGBA, cx, cy, half int, col color.Color) {
	for y := cy - half; y <= cy+half; y++ {
		for x := cx - half; x <= cx+half; x++ {
			setPixel(img, x, y, col)
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
