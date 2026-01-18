package clippoly

import (
	"fmt"
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
		{X: -2, Y: 5, Z: 0},
		{X: 2, Y: 0, Z: 0},
		{X: 2, Y: 2, Z: 0},
	}
	clipFrame := Polygon2D{
		{X: -1, Y: -1},
		{X: 3, Y: -1},
		{X: 3, Y: 3},
		{X: -1, Y: 3},
	}
	p := clip(face, clipFrame)

	fmt.Printf("p: %v\n", p)

	const (
		width  = 256
		height = 256
	)
	bounds := newBounds2D()
	bounds.addVec3(face)
	bounds.addPolygon2D(clipFrame)
	bounds.addVec3(p)
	mapper := newPixelMapper(width, height, 0.1, bounds)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, draw.Src)

	faceColor := color.RGBA{0, 0, 0, 255}
	for i := range face {
		a := toVec2(face[i])
		b := toVec2(face[(i+1)%len(face)])
		ax, ay := mapper.toPixel(a)
		bx, by := mapper.toPixel(b)
		drawLine(img, ax, ay, bx, by, faceColor)
	}
	frameColor := color.RGBA{0, 120, 220, 255}
	for i := range clipFrame {
		a := clipFrame[i]
		b := clipFrame.next(i)
		ax, ay := mapper.toPixel(a)
		bx, by := mapper.toPixel(b)
		drawLine(img, ax, ay, bx, by, frameColor)
	}
	resultColor := color.RGBA{220, 0, 0, 255}
	for i := range p {
		a := toVec2(p[i])
		b := toVec2(p.next(i))
		ax, ay := mapper.toPixel(a)
		bx, by := mapper.toPixel(b)
		drawLine(img, ax, ay, bx, by, resultColor)
	}

	outDir := filepath.Join(".", t.Name())
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	outPath := filepath.Join(outDir, "clip_face.png")
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

func saveFaceAndLinePNG(t *testing.T, face []Vec3, line []Vec2, filename string) {
	t.Helper()
	const (
		width  = 256
		height = 256
	)
	bounds := newBounds2D()
	bounds.addVec3(face)
	bounds.addVec2(line)
	mapper := newPixelMapper(width, height, 0.1, bounds)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, draw.Src)

	edgeColor := color.RGBA{0, 0, 0, 255}
	for i := range face {
		a := toVec2(face[i])
		b := toVec2(face[(i+1)%len(face)])
		ax, ay := mapper.toPixel(a)
		bx, by := mapper.toPixel(b)
		drawLine(img, ax, ay, bx, by, edgeColor)
	}
	lineColor := color.RGBA{220, 0, 0, 255}
	l0x, l0y := mapper.toPixel(line[0])
	l1x, l1y := mapper.toPixel(line[1])
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
	bounds := newBounds2D()
	bounds.addVec3(face)
	bounds.addPolygon2D(clipFrame)
	mapper := newPixelMapper(width, height, 0.1, bounds)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, draw.Src)

	faceColor := color.RGBA{0, 0, 0, 255}
	for i := range face {
		a := toVec2(face[i])
		b := toVec2(face[(i+1)%len(face)])
		ax, ay := mapper.toPixel(a)
		bx, by := mapper.toPixel(b)
		drawLine(img, ax, ay, bx, by, faceColor)
	}
	frameColor := color.RGBA{0, 120, 220, 255}
	for i := range clipFrame {
		a := clipFrame[i]
		b := clipFrame.next(i)
		ax, ay := mapper.toPixel(a)
		bx, by := mapper.toPixel(b)
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

type bounds2D struct {
	minX float64
	minY float64
	maxX float64
	maxY float64
}

func newBounds2D() bounds2D {
	return bounds2D{
		minX: math.Inf(1),
		minY: math.Inf(1),
		maxX: math.Inf(-1),
		maxY: math.Inf(-1),
	}
}

func (b *bounds2D) addVec3(points []Vec3) {
	for i := range points {
		v := points[i]
		if v.X < b.minX {
			b.minX = v.X
		}
		if v.Y < b.minY {
			b.minY = v.Y
		}
		if v.X > b.maxX {
			b.maxX = v.X
		}
		if v.Y > b.maxY {
			b.maxY = v.Y
		}
	}
}

func (b *bounds2D) addVec2(points []Vec2) {
	for i := range points {
		v := points[i]
		if v.X < b.minX {
			b.minX = v.X
		}
		if v.Y < b.minY {
			b.minY = v.Y
		}
		if v.X > b.maxX {
			b.maxX = v.X
		}
		if v.Y > b.maxY {
			b.maxY = v.Y
		}
	}
}

func (b *bounds2D) addPolygon2D(points Polygon2D) {
	for i := range points {
		v := points[i]
		if v.X < b.minX {
			b.minX = v.X
		}
		if v.Y < b.minY {
			b.minY = v.Y
		}
		if v.X > b.maxX {
			b.maxX = v.X
		}
		if v.Y > b.maxY {
			b.maxY = v.Y
		}
	}
}

type pixelMapper struct {
	minX   float64
	maxY   float64
	scale  float64
	width  int
	height int
}

func newPixelMapper(width, height int, padFrac float64, bounds bounds2D) pixelMapper {
	spanX := bounds.maxX - bounds.minX
	spanY := bounds.maxY - bounds.minY
	if spanX == 0 {
		spanX = 1
	}
	if spanY == 0 {
		spanY = 1
	}
	padX := spanX * padFrac
	padY := spanY * padFrac
	bounds.minX -= padX
	bounds.maxX += padX
	bounds.minY -= padY
	bounds.maxY += padY
	spanX = bounds.maxX - bounds.minX
	spanY = bounds.maxY - bounds.minY
	scaleX := float64(width-1) / spanX
	scaleY := float64(height-1) / spanY
	scale := math.Min(scaleX, scaleY)
	return pixelMapper{
		minX:   bounds.minX,
		maxY:   bounds.maxY,
		scale:  scale,
		width:  width,
		height: height,
	}
}

func (m pixelMapper) toPixel(p Vec2) (int, int) {
	x := int(math.Round((p.X - m.minX) * m.scale))
	y := int(math.Round((m.maxY - p.Y) * m.scale))
	if x < 0 {
		x = 0
	} else if x >= m.width {
		x = m.width - 1
	}
	if y < 0 {
		y = 0
	} else if y >= m.height {
		y = m.height - 1
	}
	return x, y
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
