package clip

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

func saveEdgesPNG(path string, edges [][]*node, nodes ...*node) error {
	return saveEdgesPNGWithHighlight(path, edges, nil, nodes...)
}

func saveEdgesPNGWithHighlight(path string, edges [][]*node, highlight [][]*node, nodes ...*node) error {
	if len(edges) == 0 && len(highlight) == 0 {
		return nil
	}

	validEdges := make([][]*node, 0, len(edges))
	highlightEdges := make([][]*node, 0, len(highlight))
	coords := make([][]Coord, 0, len(edges)+len(highlight))

	addEdges := func(src [][]*node, dst *[][]*node) {
		for _, edge := range src {
			if len(edge) < 2 || edge[0] == nil || edge[1] == nil {
				continue
			}
			*dst = append(*dst, edge)
			coords = append(coords, []Coord{edge[0].coord, edge[1].coord})
		}
	}

	addEdges(edges, &validEdges)
	addEdges(highlight, &highlightEdges)

	if len(coords) == 0 {
		return nil
	}

	minX, maxX, minY, maxY, ok := coordBounds(coords...)
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

	total := len(validEdges)
	for i, edge := range validEdges {
		x0, y0 := project(edge[0].coord)
		x1, y1 := project(edge[1].coord)
		drawLineRGBA(img, x0, y0, x1, y1, edgeColor(i, total))
	}

	if len(highlightEdges) > 0 {
		highlightColor := color.RGBA{R: 0, G: 200, B: 200, A: 255}
		for _, edge := range highlightEdges {
			x0, y0 := project(edge[0].coord)
			x1, y1 := project(edge[1].coord)
			drawLineRGBA(img, x0, y0, x1, y1, highlightColor)
		}
	}

	if len(nodes) > 0 {
		insideColor := color.RGBA{R: 30, G: 80, B: 220, A: 255}
		outsideColor := color.RGBA{R: 220, G: 40, B: 40, A: 255}
		for _, n := range nodes {
			if n == nil {
				continue
			}
			x, y := project(n.coord)
			if n.isInside {
				drawPointRGBA(img, x, y, insideColor)
			} else {
				drawPointRGBA(img, x, y, outsideColor)
			}
		}
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

func edgeColor(i, total int) color.RGBA {
	if total <= 1 {
		return color.RGBA{R: 30, G: 80, B: 220, A: 255}
	}
	hue := (float64(i) / float64(total)) * 360.0
	return hsvToRGBA(hue, 0.8, 0.95)
}

func hsvToRGBA(h, s, v float64) color.RGBA {
	h = math.Mod(h, 360.0)
	if h < 0 {
		h += 360.0
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var r1, g1, b1 float64
	switch {
	case h < 60.0:
		r1, g1, b1 = c, x, 0
	case h < 120.0:
		r1, g1, b1 = x, c, 0
	case h < 180.0:
		r1, g1, b1 = 0, c, x
	case h < 240.0:
		r1, g1, b1 = 0, x, c
	case h < 300.0:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}

	r := uint8(math.Round((r1 + m) * 255))
	g := uint8(math.Round((g1 + m) * 255))
	b := uint8(math.Round((b1 + m) * 255))
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
