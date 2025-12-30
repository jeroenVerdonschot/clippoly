package clippoly

import (
	"fmt"
	"image/color"
	"math"
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
			poly, err := Clip(tri, crop)

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
		{coord: Coord{0, 0, 0}},
		{coord: Coord{10, 0, 10}},
	}
	edge2 := []*node{
		{coord: Coord{5, -5, 0}},
		{coord: Coord{5, 5, 0}},
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

func TestClipMeshFaces(t *testing.T) {
	vertices := []Coord{
		{0, 0, 0},
		{4, 0, 0},
		{4, 4, 0},
		{0, 4, 0},
	}
	faces := [][3]int{
		{0, 1, 2},
		{0, 2, 3},
	}

	clip := Polygon{
		{2, -1, 0},
		{5, -1, 0},
		{5, 3, 0},
		{2, 3, 0},
	}

	expectedAreas := []float64{5.5, 0.5}
	area := func(poly Polygon) float64 {
		if len(poly) < 3 {
			return 0
		}
		var sum float64
		for i := range poly {
			j := (i + 1) % len(poly)
			sum += float64(poly[i][0])*float64(poly[j][1]) - float64(poly[j][0])*float64(poly[i][1])
		}
		return math.Abs(sum) * 0.5
	}

	for idx, face := range faces {
		poly := Polygon{
			vertices[face[0]],
			vertices[face[1]],
			vertices[face[2]],
		}
		clipped, err := Clip(poly, clip)
		if err != nil {
			t.Fatalf("clip face %d: %v", idx, err)
		}
		if clipped == nil {
			t.Fatalf("clip face %d: expected intersection", idx)
		}

		var total float64
		for _, tri := range clipped {
			total += area(tri)
		}

		if diff := math.Abs(total - expectedAreas[idx]); diff > 1e-3 {
			t.Fatalf("clipped area mismatch for face %d: got %.3f, want %.3f", idx, total, expectedAreas[idx])
		}

		filename := filepath.Join("test_output", fmt.Sprintf("mesh_face_%02d.png", idx+1))
		if err := saveTriangleCropPNG(filename, clip, poly, clipped); err != nil {
			t.Fatalf("save png for face %d: %v", idx, err)
		}
	}
}

func Test_meshWithReturnNewMesh(t *testing.T) {
	vertices := []Coord{
		{0, 0, 0},
		{4, 0, 4},
		{4, 4, 4},
		{0, 4, 0},
	}
	faces := [][3]int{
		{0, 1, 2},
		{0, 2, 3},
	}

	clip := Polygon{
		{2, -1, 0},
		{5, -1, 0},
		{5, 3, 0},
		{2, 3, 0},
	}

	newVerts, newFaces, err := ClipMesh(vertices, faces, clip)
	if err != nil {
		t.Fatalf("clip mesh: %v", err)
	}

	if len(newVerts) != 6 {
		t.Fatalf("expected 6 vertices after clipping, got %d", len(newVerts))
	}
	if len(newFaces) != 4 {
		t.Fatalf("expected 4 faces after clipping, got %d", len(newFaces))
	}

	expectedVerts := map[Coord]struct{}{
		{2, 0, 2}:   {},
		{4, 0, 4}:   {},
		{4, 3, 1.5}: {},
		{3, 3, 3}:   {},
		{2, 2, 2}:   {},
		{2, 3, 0}:   {},
	}
	for _, v := range newVerts {
		if _, ok := expectedVerts[v]; !ok {
			t.Fatalf("unexpected vertex in result: %v", v)
		}
	}

	area := func(poly Polygon) float64 {
		if len(poly) < 3 {
			return 0
		}
		var sum float64
		for i := range poly {
			j := (i + 1) % len(poly)
			sum += float64(poly[i][0])*float64(poly[j][1]) - float64(poly[j][0])*float64(poly[i][1])
		}
		return math.Abs(sum) * 0.5
	}

	var totalArea float64
	for idx, face := range newFaces {
		for _, vi := range face {
			if vi < 0 || vi >= len(newVerts) {
				t.Fatalf("face %d has invalid vertex index %d", idx, vi)
			}
		}
		poly := Polygon{
			newVerts[face[0]],
			newVerts[face[1]],
			newVerts[face[2]],
		}
		totalArea += area(poly)
	}

	if diff := math.Abs(totalArea - 6.0); diff > 1e-3 {
		t.Fatalf("clipped mesh area mismatch: got %.3f, want 6.000", totalArea)
	}

	filename := filepath.Join("test_output", "mesh_clip.png")
	if err := saveMeshClipPNG(filename, vertices, faces, clip, newVerts, newFaces); err != nil {
		t.Fatalf("save mesh png: %v", err)
	}
}
