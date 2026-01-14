package clippoly

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"path/filepath"
	"testing"
)

func TestMultipleTriangleCropsWithPalette_newClip(t *testing.T) {
	crop := Polygon{{0, 0}, {40, 0}, {40, 30}, {0, 30}}

	// 3 and 7 give misktake

	triangles := []Polygon{
		{{-10, -10}, {20, 10}, {10, 30}}, // bottom-left corner, edge case 1
		{{-10, -10}, {20, 10}, {0, 30}},  // edge case 2
		{{-10, 10}, {20, 10}, {10, 25}},  // left edge 3
		{{10, -5}, {30, 10}, {20, 20}},   // bottom edge 4
		{{-10, 20}, {30, 40}, {20, 15}},  // top-left edge 5
		{{10, 20}, {30, 40}, {20, 15}},   // top edge !!!! 6
		{{45, 10}, {20, 25}, {35, 35}},   // right edge !!!! 7
		{{35, -5}, {50, 15}, {25, 20}},   // bottom-right corner !!!! 8
		{{-10, 25}, {10, 40}, {20, 10}},  // left-top corner 9
		{{15, 5}, {45, 20}, {25, 35}},    // right-top corner 10
		{{5, 35}, {30, 20}, {-10, 20}},   // top overshoot !!!! 11
		{{20, 10}, {50, 20}, {30, 40}},   // right overshoot 12
		{{-5, -2}, {15, 10}, {25, -8}},   // bottom overshoot !!! 13
		{{38, 2}, {50, 25}, {20, 32}},    // right-top slice 14
		{{5, 5}, {35, 5}, {20, 25}},      // fully inside crop 15
		{{0, 0}, {40, 0}, {0, 30}},       // along crop edges 16
		{{-20, -20}, {80, 5}, {10, 60}},  // large coverage 17
		{{20, -15}, {60, 15}, {20, 55}},  // tall slice through crop 18
		{{39, 29}, {80, 29}, {39, 80}},   // tiny corner overlap 19
		{{-15, 12}, {15, 32}, {30, -8}},  // left lean slice 20
		{{18, -8}, {22, 42}, {55, 18}},   // thin vertical through center 21
		{{5, 28}, {35, 28}, {20, 55}},    // top band 22

	}

	targetColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}  // yellow
	cropColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}        // black
	highlightColor = color.RGBA{R: 255, G: 0, B: 0, A: 255} // red

	for i, tri := range triangles {
		idx := i

		t.Run(fmt.Sprintf("triangle_%02d", idx+1), func(t *testing.T) {
			poly, err := newClip(tri, crop)

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

func TestRelinkSameCoordsAsFrom(t *testing.T) {
	from := &node{coord: Coord{1, 2, 3}, id: 1}
	to := &node{coord: Coord{4, 5, 6}, id: 2}
	cross1 := &node{coord: Coord{7, 8, 9}, id: 3}
	cross2 := &node{coord: Coord{10, 11, 12}, id: 4}
	newNode := &node{coord: Coord{1, 2, 3}, id: 5}

	from.nodes = []*node{to}
	to.nodes = []*node{from}
	cross1.nodes = []*node{cross2}
	cross2.nodes = []*node{cross1}

	relink(newNode, from, to, cross1, cross2)

	if len(from.nodes) != 1 || !nodeContains(from.nodes, newNode) {
		t.Fatalf("from should link to new node only")
	}
	if nodeContains(from.nodes, to) {
		t.Fatalf("from should not link to to node after relink")
	}
	if nodeContains(to.nodes, from) {
		t.Fatalf("to should not link back to from after relink")
	}
	if len(cross1.nodes) != 1 || !nodeContains(cross1.nodes, newNode) {
		t.Fatalf("cross1 should link to new node only")
	}
	if len(cross2.nodes) != 1 || !nodeContains(cross2.nodes, newNode) {
		t.Fatalf("cross2 should link to new node only")
	}
	if nodeContains(cross1.nodes, cross2) || nodeContains(cross2.nodes, cross1) {
		t.Fatalf("cross links should be removed")
	}
	if len(newNode.nodes) != 3 ||
		!nodeContains(newNode.nodes, to) ||
		!nodeContains(newNode.nodes, cross1) ||
		!nodeContains(newNode.nodes, cross2) {
		t.Fatalf("new node links incorrect")
	}
}

func nodeContains(nodes []*node, target *node) bool {
	for _, n := range nodes {
		if n == target {
			return true
		}
	}
	return false
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

func TestClipMesh_RemovesNonIntersectingFaces(t *testing.T) {
	vertices := []Coord{
		{82.453125, -334.59375, 14.700125},
		{86.828125, -347.40625, 14.700125},
		{91.328125, -327.0625, 14.700125},
		{114.859375, -320.0625, 14.700125},
	}
	faces := [][3]int{
		{0, 1, 2},
		{3, 2, 1},
	}
	clip := Polygon{
		{0, -305.21875, 0},
		{344.89062, -305.21875, 0},
		{344.89062, 0, 0},
		{0, 0, 0},
	}

	clippedVerts, clippedFaces, err := ClipMesh(vertices, faces, clip)
	if err != nil {
		t.Fatalf("clip mesh: %v", err)
	}
	if len(clippedVerts) != 0 {
		t.Fatalf("expected no vertices after clipping, got %d", len(clippedVerts))
	}
	if len(clippedFaces) != 0 {
		t.Fatalf("expected no faces after clipping, got %d", len(clippedFaces))
	}

	filename := filepath.Join("test_output", "mesh_clip_bug.png")
	if err := saveMeshClipPNG(filename, vertices, faces, clip, clippedVerts, clippedFaces); err != nil {
		t.Fatalf("save mesh png: %v", err)
	}
}

func TestClip_TriangleNearEdge(t *testing.T) {
	tri := Polygon{
		{1, 0, 0},
		{2, -1, 0},
		{2, 1, 0},
	}
	clip := Polygon{
		{0, -3, 0},
		{3, -3, 0},
		{3, 0, 0},
		{0, 0, 0},
	}

	clipped, err := Clip(tri, clip)
	if err != nil {
		t.Fatalf("clip triangle: %v", err)
	}
	if clipped == nil {
		t.Fatalf("expected clipped triangle, got nil")
	}

	filename := filepath.Join("test_output", "triangle_clip_edge.png")
	if err := saveTriangleCropPNG(filename, clip, tri, clipped); err != nil {
		t.Fatalf("save png: %v", err)
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name string
		a1   Coord
		a2   Coord
		b1   Coord
		b2   Coord
		want bool
	}{

		{
			name: "endpoint_no_intersection",
			a1:   Coord{1, 1, 0},
			a2:   Coord{1, 3, 0},
			b1:   Coord{0, 0, 0},
			b2:   Coord{2, 0, 0},
			want: false,
		},
		{
			name: "endpoint_intersection_endpoint",
			a1:   Coord{1, 0, 0},
			a2:   Coord{1, 2, 0},
			b1:   Coord{0, 0, 0},
			b2:   Coord{2, 0, 0},
			want: false,
		},
		{
			name: "proper_crossing_interior",
			a1:   Coord{1, -1, 0},
			a2:   Coord{1, 1, 0},
			b1:   Coord{0, 0, 0},
			b2:   Coord{2, 0, 0},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// if got := segmentsIntersect(tt.a1, tt.a2, tt.b1, tt.b2); got != tt.want {
			// 	t.Fatalf("segmentsIntersect(%v, %v, %v, %v) = %v, want %v", tt.a1, tt.a2, tt.b1, tt.b2, got, tt.want)
			// }
			edge1 := []*node{
				{coord: tt.a1},
				{coord: tt.a2},
			}
			edge2 := []*node{
				{coord: tt.b1},
				{coord: tt.b2},
			}
			intersect := findIntersect(edge1, edge2)
			if got := intersect != nil; got != tt.want {
				t.Fatalf("findIntersect(%v, %v, %v, %v) != nil = %v, want %v", tt.a1, tt.a2, tt.b1, tt.b2, got, tt.want)
			}
		})
	}

	for _, tt := range tests {
		filename := filepath.Join("test_output", fmt.Sprintf("intersect_%s.png", sanitizeFilename(tt.name)))
		if err := saveIntersectPNG(filename, tt.a1, tt.a2, tt.b1, tt.b2); err != nil {
			t.Fatalf("save png: %v", err)
		}
	}
}
func TestPointOfSegemtn(t *testing.T) {
	tests := []struct {
		name string
		p    Coord
		a2   Coord
		b1   Coord
		b2   Coord
		want bool
	}{
		{
			name: "endpoint_intersection_excluded",
			p:    Coord{1, 0, 0},
			b1:   Coord{0, 0, 0},
			b2:   Coord{2, 0, 0},
			want: true,
		},
		{
			name: "proper_crossing_interior",
			p:    Coord{1, -1, 0},
			b1:   Coord{0, 0, 0},
			b2:   Coord{2, 0, 0},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pointOnEdge(tt.p[0], tt.p[1], tt.b1[0], tt.b1[1], tt.b2[0], tt.b2[1]); got != tt.want {
				t.Fatalf("segmentsIntersect(%v, %v, %v, %v) = %v, want %v", tt.p, tt.a2, tt.b1, tt.b2, got, tt.want)
			}
		})
	}

	for _, tt := range tests {
		filename := filepath.Join("test_output", fmt.Sprintf("intersect_%s.png", sanitizeFilename(tt.name)))
		if err := saveIntersectPNG(filename, tt.p, tt.a2, tt.b1, tt.b2); err != nil {
			t.Fatalf("save png: %v", err)
		}
	}
}

func Test_newClip(t *testing.T) {
	// // {{-10, -10}, {20, 10}, {0, 30}}, // edge case 2
	// tri := Polygon{
	// 	{-10, -10}, {20, 10}, {0, 30}, // edge case 2
	// }
	// clip := Polygon{{0, 0}, {40, 0}, {40, 30}, {0, 30}}

	tri := Polygon{
		{1, 0, 0},
		{2, -1, 0},
		{2, 1, 0},
	}
	clip := Polygon{
		{0, -3, 0},
		{3, -3, 0},
		{3, 0, 0},
		{0, 0, 0},
	}
	idGen := &idGenerator{}

	targetNodes := makeShapeWithID(tri, true, idGen)
	clipNodes := makeShapeWithID(clip, false, idGen)

	areAllInside := setIsInside(targetNodes, clipNodes)
	if areAllInside {
		// triangulte
	}
	areAllInside = setIsInside(clipNodes, targetNodes)
	if areAllInside {
		// triangulte
	}

	clipEdges := edges(clipNodes)
	targetNodes, clipEdges = intersectPointOnEdge(targetNodes, clipEdges)

	targetEdges := edges(targetNodes)
	targetEdges, clipEdges = intersect(targetEdges, clipEdges, idGen)

	allEdges := make([][]*node, 0, len(targetEdges)+len(clipEdges))
	allEdges = append(allEdges, targetEdges...)
	allEdges = append(allEdges, clipEdges...)

	// TEMP

	allRelevant := make([][]*node, 0, len(allEdges))
	for _, e := range allEdges {
		if e[0].isInside && e[1].isInside {
			allRelevant = append(allRelevant, e)
		}
	}

	adjMap := make(map[*node][]*node)
	for _, edge := range allRelevant {
		adjMap[edge[0]] = append(adjMap[edge[0]], edge[1])
		adjMap[edge[1]] = append(adjMap[edge[1]], edge[0])
	}

	for k, v := range adjMap {
		fmt.Printf("k.id: %v\n", k.id)
		for i, c := range v {
			fmt.Printf("v%v: %v\n", i, c.id)
		}
	}

	// startNode := allRelevant[0][0]
	// loop := []*node{}
	// curr := adjMap[startNode][0]
	// loop = append(loop, curr)
	// next := adjMap[curr][0]
	// adjMap[curr] = nil
	// for range len(adjMap) {
	// }

	allNodes := make([]*node, 0, len(targetNodes)+len(clipNodes))
	seen := make(map[*node]struct{}, len(targetNodes)+len(clipNodes))
	addNode := func(n *node) {
		if n == nil {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		allNodes = append(allNodes, n)
	}
	for _, n := range targetNodes {
		addNode(n)
	}
	for _, n := range clipNodes {
		addNode(n)
	}
	for _, edge := range targetEdges {
		for _, n := range edge {
			addNode(n)
		}
	}
	for _, edge := range clipEdges {
		for _, n := range edge {
			addNode(n)
		}
	}

	filename := filepath.Join("test_output", "newclip_edges.png")
	if err := saveEdgesPNGWithHighlight(filename, allEdges, allRelevant, allNodes...); err != nil {
		t.Fatalf("save edges png: %v", err)
	}

	// allIntersections

}
func Test_newClip_2(t *testing.T) {

	tri := Polygon{
		{2, 1, 0},
		{1, 0, 0},
		{2, -2, 0},
	}
	clip := Polygon{
		{0, -3, 0},
		{3, -3, 0},
		{3, 0, 0},
		{0, 0, 0},
	}

	ps, _ := newClip(tri, clip)

	filename := filepath.Join("test_output", "newclip_triangles.png")
	if err := saveTriangleCropPNG(filename, clip, tri, ps); err != nil {
		t.Fatalf("save png: %v", err)
	}

}

func newClip(tri, clip Polygon) (Polygons, error) {

	idGen := &idGenerator{}

	targetNodes := makeShapeWithID(tri, true, idGen)
	clipNodes := makeShapeWithID(clip, false, idGen)

	areAllInside := setIsInside(targetNodes, clipNodes)
	if areAllInside {
		return triangulate(targetNodes)
	}
	areAllInside = setIsInside(clipNodes, targetNodes)
	if areAllInside {
		return triangulate(clipNodes)
	}

	mergeCoincidentNodes(targetNodes, clipNodes)

	clipEdges := edges(clipNodes)
	targetNodes, clipEdges = intersectPointOnEdge(targetNodes, clipEdges)

	targetEdges := edges(targetNodes)
	targetEdges, clipEdges = intersect(targetEdges, clipEdges, idGen)

	allEdges := make([][]*node, 0, len(targetEdges)+len(clipEdges))
	allEdges = append(allEdges, targetEdges...)
	allEdges = append(allEdges, clipEdges...)

	// TEMP

	allRelevant := make([][]*node, 0, len(allEdges))
	for _, e := range allEdges {
		if e[0].isInside && e[1].isInside {
			allRelevant = append(allRelevant, e)
		}
	}

	adjMap := make(map[*node][]*node)
	for _, edge := range allRelevant {
		adjMap[edge[0]] = append(adjMap[edge[0]], edge[1])
		adjMap[edge[1]] = append(adjMap[edge[1]], edge[0])
	}

	// Build the loop starting from first node
	loop := make([]*node, 0, len(allRelevant)+1)
	loop = append(loop, allRelevant[0][0])
	prev := allRelevant[0][0]
	current := allRelevant[0][1]

	// todo add safety max len(target)*len(clip)*2 iterations
	for current != loop[0] {
		loop = append(loop, current)

		// Find next node (the neighbor that isn't prev)
		neighbors := adjMap[current]
		var next *node
		for _, neighbor := range neighbors {
			if neighbor != prev {
				next = neighbor
				break
			}
		}

		prev = current
		current = next
	}

	if len(loop) != len(allRelevant) {
		log.Fatalf("loop incomplete: visited %d nodes but have %d edges", len(loop), len(allRelevant))
	}

	return triangulate(loop)

}

func mergeCoincidentNodes(targetNodes, clipNodes []*node) {
	targetByCoord := make(map[Coord]*node, len(targetNodes))
	for _, tn := range targetNodes {
		targetByCoord[tn.coord] = tn
	}
	for i, cn := range clipNodes {
		tn, ok := targetByCoord[cn.coord]
		if !ok || tn == cn {
			continue
		}
		clipNodes[i] = tn
		for _, nb := range cn.nodes {
			if nb == nil || nb == tn {
				continue
			}
			nb.remove(cn)
			if !nodeContains(nb.nodes, tn) {
				nb.add(tn)
			}
			if !nodeContains(tn.nodes, nb) {
				tn.add(nb)
			}
		}
		tn.isInside = tn.isInside || cn.isInside
		cn.nodes = nil
	}
}

func intersectPointOnEdge(targetNodes []*node, clip [][]*node) ([]*node, [][]*node) {

	// check is targe node lay on a clip edge (pointonedge)
	// if so add node to target and to clip
	// set isInside true

	for _, tn := range targetNodes {
		px := tn.coord[0]
		py := tn.coord[1]
		for i := 0; i < len(clip); i++ {
			edge := clip[i]
			if len(edge) < 2 || edge[0] == nil || edge[1] == nil {
				continue
			}
			a := edge[0]
			b := edge[1]
			if tn.coord == a.coord || tn.coord == b.coord {
				tn.isInside = true
				continue
			}
			if !pointOnEdge(px, py, a.coord[0], a.coord[1], b.coord[0], b.coord[1]) {
				continue
			}

			tn.isInside = true
			clip[i] = []*node{a, tn}
			clip = append(clip, []*node{tn, b})

			a.remove(b)
			b.remove(a)
			a.add(tn)
			b.add(tn)
			tn.add(a)
			tn.add(b)
		}
	}

	return targetNodes, clip

}

func intersect(target, clip [][]*node, id *idGenerator) ([][]*node, [][]*node) {

	for i := 0; i < len(clip); i++ {
		edge1 := clip[i]

		for j := 0; j < len(target); j++ {
			edge2 := target[j]

			intNode := findIntersect(edge1, edge2)

			if intNode != nil {

				intNode.id = id.Next()

				relink(intNode, edge1[0], edge1[1], edge2[0], edge2[1])

				edge1End := edge1[1]
				edge2End := edge2[1]

				edge1[1] = intNode
				clip[i] = edge1
				clip = append(clip, []*node{intNode, edge1End})
				edge2[1] = intNode
				target[j] = edge2
				target = append(target, []*node{intNode, edge2End})
				// reset for loop
				i = -1
				break
			}

		}

	}

	return target, clip

}

func edges(nodes []*node) [][]*node {

	ln := len(nodes)
	if ln < 2 {
		return nil
	}

	list := make([][]*node, 0, ln)
	for i := 0; i < ln; i++ {
		next := i + 1
		if next == ln {
			next = 0
		}
		list = append(list, []*node{nodes[i], nodes[next]})
	}

	return list

}

func Test_edges(t *testing.T) {

	n1 := &node{id: 1}
	n2 := &node{id: 2}
	n3 := &node{id: 3}

	got := edges([]*node{n1, n2, n3})
	want := [][]*node{
		{n1, n2},
		{n2, n3},
		{n3, n1},
	}

	if len(got) != len(want) {
		t.Fatalf("edges length = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if len(got[i]) != 2 {
			t.Fatalf("edge %d length = %d, want 2", i, len(got[i]))
		}
		if got[i][0] != want[i][0] || got[i][1] != want[i][1] {
			t.Fatalf("edge %d = %v, want %v", i, got[i], want[i])
		}
	}

}
