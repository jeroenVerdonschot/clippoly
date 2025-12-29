package clippoly

import (
	"fmt"
	"image/color"
	"path/filepath"
	"testing"
)

func TestMultipleTriangleCropsWithPalette(t *testing.T) {
	crop := Polygon{{0, 0}, {40, 0}, {40, 30}, {0, 30}}

	// 3 and 7 give misktake

	triangles := []Polygon{
		{{-10, 10}, {20, 10}, {10, 25}},  // left edge
		{{10, -5}, {30, 10}, {20, 20}},   // bottom edge
		{{-10, 20}, {30, 40}, {20, 15}},  // top-left edge
		{{10, 20}, {30, 40}, {20, 15}},   // top edge !!!!
		{{45, 10}, {20, 25}, {35, 35}},   // right edge !!!!
		{{-10, -10}, {20, 10}, {10, 30}}, // bottom-left corner
		{{35, -5}, {50, 15}, {25, 20}},   // bottom-right corner !!!!
		{{-10, 25}, {10, 40}, {20, 10}},  // left-top corner
		{{15, 5}, {45, 20}, {25, 35}},    // right-top corner
		{{5, 35}, {30, 20}, {-10, 20}},   // top overshoot !!!!
		{{20, 10}, {50, 20}, {30, 40}},   // right overshoot
		{{-5, -2}, {15, 10}, {25, -8}},   // bottom overshoot !!!
		{{38, 2}, {50, 25}, {20, 32}},    // right-top slice

		{{5, 5}, {35, 5}, {20, 25}},     // fully inside crop
		{{0, 0}, {40, 0}, {0, 30}},      // along crop edges
		{{-20, -20}, {80, 5}, {10, 60}}, // large coverage
		{{20, -15}, {60, 15}, {20, 55}}, // tall slice through crop
		{{39, 29}, {80, 29}, {39, 80}},  // tiny corner overlap

		{{-15, 12}, {15, 32}, {30, -8}}, // left lean slice
		{{18, -8}, {22, 42}, {55, 18}},  // thin vertical through center
		{{5, 28}, {35, 28}, {20, 55}},   // top band

	}

	targetColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}  // yellow
	cropColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}        // black
	highlightColor = color.RGBA{R: 255, G: 0, B: 0, A: 255} // red

	for i, tri := range triangles {
		idx := i

		t.Run(fmt.Sprintf("triangle_%02d", idx+1), func(t *testing.T) {
			poly, err := Crop(tri, crop)

			if err != nil {
				t.Fatalf("crop failed: %v", err)
			}
			if poly == nil {
				t.Fatalf("expected cropped triangles for triangle_%02d", idx+1)
			}

			filename := filepath.Join("test_output", fmt.Sprintf("triangle_%02d.png", idx+1))
			if err := saveTriangleCropPNG(filename, crop, tri, poly); err != nil {
				t.Fatalf("save png: %v", err)
			}

		})
	}
}
func TestFindIntersectInterpolatesZ(t *testing.T) {
	edge1 := []*node{
		{coord: coord{0, 0, 0}},
		{coord: coord{10, 0, 10}},
	}
	edge2 := []*node{
		{coord: coord{5, -5, 0}},
		{coord: coord{5, 5, 0}},
	}

	intersection := findIntersect(edge1, edge2)
	if intersection == nil {
		t.Fatalf("expected intersection, got nil")
	}

	const eps = 1e-4
	if dx := float64(intersection.coord[0] - 5); dx < -eps || dx > eps {
		t.Fatalf("unexpected x: got %v, want ~5", intersection.coord[0])
	}
	if dy := float64(intersection.coord[1]); dy < -eps || dy > eps {
		t.Fatalf("unexpected y: got %v, want ~0", intersection.coord[1])
	}
	if dz := float64(intersection.coord[2] - 5); dz < -eps || dz > eps {
		t.Fatalf("unexpected z interpolation: got %v, want ~5", intersection.coord[2])
	}
}
