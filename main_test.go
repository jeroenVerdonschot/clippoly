package clippoly

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func almostEqual(a, b float64) bool {
	if a > b {
		return a-b <= eps*10
	}
	return b-a <= eps*10
}

func assertVec3(t *testing.T, got, want Vec3) {
	t.Helper()
	if !almostEqual(got.X, want.X) || !almostEqual(got.Y, want.Y) || !almostEqual(got.Z, want.Z) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestIntersectBasic(t *testing.T) {
	edge := []Vec3{{X: 0, Y: 0, Z: 0}, {X: 10, Y: 0, Z: 0}}
	cut := []Vec2{{X: 5, Y: -5}, {X: 5, Y: 5}}
	got := intersect(edge, cut)
	assertVec3(t, got, Vec3{X: 5, Y: 0, Z: 0})
}

func TestIntersectParallel(t *testing.T) {
	edge := []Vec3{{X: 0, Y: 0, Z: 0}, {X: 10, Y: 0, Z: 0}}
	cut := []Vec2{{X: 0, Y: 1}, {X: 10, Y: 1}}
	got := intersect(edge, cut)
	assertVec3(t, got, Vec3{})
}

func TestIntersectClamp(t *testing.T) {
	edge := []Vec3{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 0, Z: 0}}
	cut := []Vec2{{X: 2, Y: -1}, {X: 2, Y: 1}}
	got := intersect(edge, cut)
	assertVec3(t, got, Vec3{X: 1, Y: 0, Z: 0})
}

func TestIntersectZInterpolation(t *testing.T) {
	edge := []Vec3{{X: 0, Y: 0, Z: 0}, {X: 10, Y: 0, Z: 10}}
	cut := []Vec2{{X: 5, Y: -2}, {X: 5, Y: 2}}
	got := intersect(edge, cut)
	assertVec3(t, got, Vec3{X: 5, Y: 0, Z: 5})
}

func TestIntersectOverlapping(t *testing.T) {
	edge := []Vec3{{X: 0, Y: 0, Z: 0}, {X: 10, Y: 0, Z: 0}}
	cut := []Vec2{{X: -5, Y: 0}, {X: 15, Y: 0}}
	got := intersect(edge, cut)
	assertVec3(t, got, Vec3{})
}

func TestSliceFaceReturnsNil(t *testing.T) {
	face := []Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 2, Y: 0, Z: 0},
		{X: 0, Y: 2, Z: 0},
	}
	line := []Vec2{{X: 1, Y: -1}, {X: 1, Y: 3}}
	got, err := sliceFace(face, line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil slice, got %+v", got)
	}
	saveFaceAndLinePNG(t, face, line, "slice_face.png")
}

func TestClipPrintsLines(t *testing.T) {
	face := []Vec3{
		{X: 12, Y: 0, Z: 0},
		{X: 2, Y: 0, Z: 0},
		{X: 2, Y: 2, Z: 0},
	}
	clipFrame := Polygon2D{
		{X: -1, Y: -1},
		{X: 3, Y: -1},
		{X: 3, Y: 3},
		{X: -1, Y: 3},
	}
	clip(face, clipFrame)

	saveFaceAndClipFramePNG(t, face, clipFrame, "clip_face.png")
}

func saveFaceAndLinePNG(t *testing.T, face []Vec3, line []Vec2, filename string) {
	t.Helper()
	const (
		width  = 256
		height = 256
	)
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, v := range face {
		if v.X < minX {
			minX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}
	for _, v := range line {
		if v.X < minX {
			minX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}
	spanX := maxX - minX
	spanY := maxY - minY
	if spanX == 0 {
		spanX = 1
	}
	if spanY == 0 {
		spanY = 1
	}
	padX := spanX * 0.1
	padY := spanY * 0.1
	minX -= padX
	maxX += padX
	minY -= padY
	maxY += padY
	spanX = maxX - minX
	spanY = maxY - minY
	scaleX := float64(width-1) / spanX
	scaleY := float64(height-1) / spanY
	scale := math.Min(scaleX, scaleY)

	toPixel := func(p Vec2) (int, int) {
		x := int(math.Round((p.X - minX) * scale))
		y := int(math.Round((maxY - p.Y) * scale))
		if x < 0 {
			x = 0
		} else if x >= width {
			x = width - 1
		}
		if y < 0 {
			y = 0
		} else if y >= height {
			y = height - 1
		}
		return x, y
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, draw.Src)

	edgeColor := color.RGBA{0, 0, 0, 255}
	for i := range face {
		a := toVec2(face[i])
		b := toVec2(face[(i+1)%len(face)])
		ax, ay := toPixel(a)
		bx, by := toPixel(b)
		drawLine(img, ax, ay, bx, by, edgeColor)
	}
	lineColor := color.RGBA{220, 0, 0, 255}
	l0x, l0y := toPixel(line[0])
	l1x, l1y := toPixel(line[1])
	drawLine(img, l0x, l0y, l1x, l1y, lineColor)

	outDir := filepath.Join(".", t.Name())
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	outPath := filepath.Join(outDir, filename)
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	t.Logf("wrote %s", outPath)
}

func saveFaceAndClipFramePNG(t *testing.T, face []Vec3, clipFrame Polygon2D, filename string) {
	t.Helper()
	const (
		width  = 256
		height = 256
	)
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, v := range face {
		if v.X < minX {
			minX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}
	for _, v := range clipFrame {
		if v.X < minX {
			minX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}
	spanX := maxX - minX
	spanY := maxY - minY
	if spanX == 0 {
		spanX = 1
	}
	if spanY == 0 {
		spanY = 1
	}
	padX := spanX * 0.1
	padY := spanY * 0.1
	minX -= padX
	maxX += padX
	minY -= padY
	maxY += padY
	spanX = maxX - minX
	spanY = maxY - minY
	scaleX := float64(width-1) / spanX
	scaleY := float64(height-1) / spanY
	scale := math.Min(scaleX, scaleY)

	toPixel := func(p Vec2) (int, int) {
		x := int(math.Round((p.X - minX) * scale))
		y := int(math.Round((maxY - p.Y) * scale))
		if x < 0 {
			x = 0
		} else if x >= width {
			x = width - 1
		}
		if y < 0 {
			y = 0
		} else if y >= height {
			y = height - 1
		}
		return x, y
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, draw.Src)

	faceColor := color.RGBA{0, 0, 0, 255}
	for i := range face {
		a := toVec2(face[i])
		b := toVec2(face[(i+1)%len(face)])
		ax, ay := toPixel(a)
		bx, by := toPixel(b)
		drawLine(img, ax, ay, bx, by, faceColor)
	}
	frameColor := color.RGBA{0, 120, 220, 255}
	for i := range clipFrame {
		a := clipFrame[i]
		b := clipFrame.next(i)
		ax, ay := toPixel(a)
		bx, by := toPixel(b)
		drawLine(img, ax, ay, bx, by, frameColor)
	}

	outDir := filepath.Join(".", t.Name())
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	outPath := filepath.Join(outDir, filename)
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	t.Logf("wrote %s", outPath)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		img.Set(x0, y0, c)
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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
