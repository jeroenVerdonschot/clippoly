package clippoly

import (
	"fmt"
	"math"
)

type Coord [3]float64

type Polygon []Coord

type Polygons []Polygon

type node struct {
	coord     Coord
	isInside  bool
	nodes     []*node
	id        int
	isTarget  bool
	isVisited bool
}

type idGenerator struct {
	current int
}

func (g *idGenerator) Next() int {
	g.current++
	return g.current
}

func classifyNodes(nodes []*node, polygon []*node) bool {
	c := 0
	for _, n := range nodes {
		if isInsideNodes(n, polygon) {
			n.isInside = true
			c++
		}
	}
	return c == len(nodes)
}

func (nd *node) remove(n *node) {
	filtered := make([]*node, 0, len(nd.nodes)-1)
	for _, t := range nd.nodes {
		if t.id != n.id {
			filtered = append(filtered, t)
		}
	}
	nd.nodes = filtered
}

func (n *node) add(node *node) {
	n.nodes = append(n.nodes, node)
}

func relink(new, from, to, cross1, cross2 *node) {

	from.remove(to)
	to.remove(from)
	cross1.remove(cross2)
	cross2.remove(cross1)

	from.add(new)
	cross1.add(new)
	cross2.add(new)

	new.add(to)
	new.add(cross1)
	new.add(cross2)

}

// find all intersetions

func Clip(target, clip Polygon) (triangles Polygons, err error) {

	if len(target) < 3 {
		return nil, fmt.Errorf("target polygon must have at least 3 vertices, got %d", len(target))
	}
	if len(clip) < 3 {
		return nil, fmt.Errorf("clip polygon must have at least 3 vertices, got %d", len(clip))
	}

	// Early exit: check if polygons don't intersect at all
	if !polygonsIntersect(target, clip) { // TODO check if improvemnt
		// Check if target is completely inside or outside clip
		if isInsidePolygon(target[0], clip) {
			// Target is completely inside clip, return triangulated target
			return triangulatePolygon(target), nil
		}
		// Target is completely outside clip, return empty
		return triangulatePolygon(clip), nil
	}

	idGen := &idGenerator{}

	targetNodes := makeShapeWithID(target, true, idGen)
	clipNodes := makeShapeWithID(clip, false, idGen)

	areAllInside := classifyNodes(targetNodes, clipNodes)
	if areAllInside {
		return triangulate(targetNodes) // loop is clippoly
	}
	areAllInside = classifyNodes(clipNodes, targetNodes)
	if areAllInside {
		return triangulate(clipNodes)
	}

	loop, err := traceIntersectionLoop(targetNodes, clipNodes, idGen)
	if err != nil {
		return nil, err
	}

	return triangulate(loop)
}

// polygonsIntersect checks if two polygons have any edge intersections
func polygonsIntersect(poly1, poly2 Polygon) bool {
	// First check bounding boxes for quick rejection
	if !boundingBoxesOverlap(poly1, poly2) {
		return false
	}

	// Check if any edges intersect
	for i := 0; i < len(poly1); i++ {
		next := (i + 1) % len(poly1)
		a1, a2 := poly1[i], poly1[next]

		for j := 0; j < len(poly2); j++ {
			nextJ := (j + 1) % len(poly2)
			b1, b2 := poly2[j], poly2[nextJ]

			if segmentsIntersect(a1, a2, b1, b2) {
				return true
			}
		}
	}

	return false
}

// boundingBoxesOverlap checks if bounding boxes of two polygons overlap
func boundingBoxesOverlap(poly1, poly2 Polygon) bool {
	if len(poly1) == 0 || len(poly2) == 0 {
		return false
	}

	// Calculate bounding box for poly1
	min1X, max1X := poly1[0][0], poly1[0][0]
	min1Y, max1Y := poly1[0][1], poly1[0][1]
	for _, p := range poly1[1:] {
		if p[0] < min1X {
			min1X = p[0]
		}
		if p[0] > max1X {
			max1X = p[0]
		}
		if p[1] < min1Y {
			min1Y = p[1]
		}
		if p[1] > max1Y {
			max1Y = p[1]
		}
	}

	// Calculate bounding box for poly2
	min2X, max2X := poly2[0][0], poly2[0][0]
	min2Y, max2Y := poly2[0][1], poly2[0][1]
	for _, p := range poly2[1:] {
		if p[0] < min2X {
			min2X = p[0]
		}
		if p[0] > max2X {
			max2X = p[0]
		}
		if p[1] < min2Y {
			min2Y = p[1]
		}
		if p[1] > max2Y {
			max2Y = p[1]
		}
	}

	// Check overlap
	return min1X < max2X && max1X > min2X && min1Y < max2Y && max1Y > min2Y
}

// segmentsIntersect checks if two line segments intersect (excluding endpoints)
func segmentsIntersect(a1, a2, b1, b2 Coord) bool {
	// Quick bounding box check
	aMinX, aMaxX := a1[0], a2[0]
	if aMinX > aMaxX {
		aMinX, aMaxX = aMaxX, aMinX
	}
	aMinY, aMaxY := a1[1], a2[1]
	if aMinY > aMaxY {
		aMinY, aMaxY = aMaxY, aMinY
	}

	bMinX, bMaxX := b1[0], b2[0]
	if bMinX > bMaxX {
		bMinX, bMaxX = bMaxX, bMinX
	}
	bMinY, bMaxY := b1[1], b2[1]
	if bMinY > bMaxY {
		bMinY, bMaxY = bMaxY, bMinY
	}

	if aMaxX < bMinX || aMinX > bMaxX || aMaxY < bMinY || aMinY > bMaxY {
		return false
	}

	// Calculate intersection
	ax := a2[0] - a1[0]
	ay := a2[1] - a1[1]
	bx := b2[0] - b1[0]
	by := b2[1] - b1[1]
	den := ax*by - ay*bx

	if den == 0 {
		return false // Parallel or collinear
	}

	cx := b1[0] - a1[0]
	cy := b1[1] - a1[1]
	t := (cx*by - cy*bx) / den
	u := (cx*ay - cy*ax) / den

	// Check if intersection is strictly between endpoints
	return t > 0 && t < 1 && u > 0 && u < 1
}

// isInsidePolygon checks if a point is inside a polygon
func isInsidePolygon(point Coord, poly Polygon) bool {
	if len(poly) < 3 {
		return false
	}

	px := float64(point[0])
	py := float64(point[1])
	inside := false

	prev := poly[len(poly)-1]
	for _, curr := range poly {
		x1 := float64(prev[0])
		y1 := float64(prev[1])
		x2 := float64(curr[0])
		y2 := float64(curr[1])

		if (y1 > py) != (y2 > py) {
			xInt := (x2-x1)*(py-y1)/(y2-y1) + x1
			if px < xInt {
				inside = !inside
			}
		}
		prev = curr
	}

	return inside
}

// triangulatePolygon is a helper to triangulate a simple polygon
func triangulatePolygon(poly Polygon) Polygons {
	if len(poly) < 3 {
		return nil
	}

	triangles := make([]Polygon, 0, len(poly)-2)
	for i := 1; i < len(poly)-1; i++ {
		triangles = append(triangles, Polygon{
			poly[0],
			poly[i],
			poly[i+1],
		})
	}

	return triangles
}

func traceIntersectionLoop(targetNodes, clipNodes []*node, idGen *idGenerator) ([]*node, error) {
	const maxIterations = 1000 // Use a more reasonable limit

	loop := make([]*node, 0, 12) // Pre-allocate with reasonable capacity
	curNode := targetNodes[0]

	for range maxIterations {

		nextNode, finished := findNextNode(curNode, loop, targetNodes, clipNodes, idGen)

		if finished {
			return loop, nil
		}

		if nextNode == nil {
			return nil, fmt.Errorf("failed to find next node in intersection loop")
		}

		loop = append(loop, nextNode)

		// fmt.Printf("nextNode.id: %v\n", nextNode.id)
		// fmt.Printf("loop: %v\n", loop)

		curNode = nextNode

	}

	return nil, fmt.Errorf("exceeded max iterations (%d) while tracing loop", maxIterations)
}

func findNextNode(curNode *node, loop []*node, targetNodes, clipNodes []*node, idGen *idGenerator) (*node, bool) {
	for _, n := range curNode.nodes {
		// Check if we've completed the loop

		if len(loop) > 0 && n.id == loop[0].id {
			return nil, true
		}

		// Check for intersections
		nodes := clipNodes
		if !n.isTarget {
			nodes = targetNodes
		}

		var intNode *node

		if intNode = checkIntersections(curNode, n, nodes, idGen); intNode != nil {
			if intNode.coord == curNode.coord {
				continue
			}
			return intNode, false
		}

		// Check if node is inside
		if n.isInside {
			return n, false
		}
	}

	return nil, false
}

func checkIntersections(curNode, n *node, nodes []*node, idGen *idGenerator) *node {
	edge1 := []*node{curNode, n}

	for _, cl := range nodes {
		for _, link := range cl.nodes {
			edge2 := []*node{cl, link}

			if intNode := findIntersect(edge1, edge2); intNode != nil {
				intNode.isInside = true
				intNode.id = idGen.Next()
				relink(intNode, curNode, n, cl, link)
				return intNode
			}

		}
	}

	return nil
}

func makeShapeWithID(poly Polygon, isTarget bool, idGen *idGenerator) []*node {
	ln := len(poly)
	if ln == 0 {
		return nil
	}

	nodes := make([]*node, ln)
	for i, c := range poly {
		nodes[i] = &node{
			coord:    c,
			id:       idGen.Next(),
			isTarget: isTarget,
		}
	}

	// Link nodes in a ring
	for i := range nodes {
		prev := (i - 1 + ln) % ln
		next := (i + 1) % ln
		nodes[i].nodes = []*node{nodes[prev], nodes[next]}
	}

	return nodes
}

func triangulate(nodes []*node) (Polygons, error) {
	ln := len(nodes)
	if ln < 3 {
		return nil, fmt.Errorf("triangulate: not enough edges (need at least 3, got %d)", ln)
	}

	triangles := make([]Polygon, 0, ln-2)

	// Fan triangulation from first vertex
	for i := 1; i < ln-1; i++ {
		triangles = append(triangles, Polygon{
			nodes[0].coord,
			nodes[i].coord,
			nodes[i+1].coord,
		})
	}

	return triangles, nil
}

func isInsideNodes(n1 *node, n2 []*node) bool {
	if n1 == nil || len(n2) < 3 {
		return false
	}
	px := float64(n1.coord[0])
	py := float64(n1.coord[1])
	inside := false
	const eps = 1e-9
	prev := n2[len(n2)-1]
	for _, curr := range n2 {
		if curr == nil || prev == nil {
			prev = curr
			continue
		}

		x1 := float64(prev.coord[0])
		y1 := float64(prev.coord[1])
		x2 := float64(curr.coord[0])
		y2 := float64(curr.coord[1])

		if n1.coord == curr.coord ||
			n1.coord == prev.coord {
			return true
		}

		if pointOnEdge(px, py, x1, y1, x2, y2) {
			return true
		}

		if math.Abs(y1-y2) < eps {
			prev = curr
			continue
		}
		if (y1 > py) != (y2 > py) {
			xInt := (x2-x1)*(py-y1)/(y2-y1) + x1
			if math.Abs(px-xInt) < eps {
				return true
			}
			if px < xInt {
				inside = !inside
			}
		}
		prev = curr
	}
	return inside
}

func findIntersect(edge1 []*node, edge2 []*node) *node {
	if len(edge1) < 2 || len(edge2) < 2 {
		return nil
	}

	a1 := edge1[0].coord
	a2 := edge1[1].coord
	b1 := edge2[0].coord
	b2 := edge2[1].coord

	// Quick reject using bounding boxes
	aMinX, aMaxX := a1[0], a2[0]
	if aMinX > aMaxX {
		aMinX, aMaxX = aMaxX, aMinX
	}
	aMinY, aMaxY := a1[1], a2[1]
	if aMinY > aMaxY {
		aMinY, aMaxY = aMaxY, aMinY
	}
	bMinX, bMaxX := b1[0], b2[0]
	if bMinX > bMaxX {
		bMinX, bMaxX = bMaxX, bMinX
	}
	bMinY, bMaxY := b1[1], b2[1]
	if bMinY > bMaxY {
		bMinY, bMaxY = bMaxY, bMinY
	}
	if aMaxX < bMinX || aMinX > bMaxX || aMaxY < bMinY || aMinY > bMaxY {
		return nil
	}

	ax := a2[0] - a1[0]
	ay := a2[1] - a1[1]
	bx := b2[0] - b1[0]
	by := b2[1] - b1[1]
	den := ax*by - ay*bx

	if den == 0 {
		return nil
	}

	cx := b1[0] - a1[0]
	cy := b1[1] - a1[1]
	t := (cx*by - cy*bx) / den
	u := (cx*ay - cy*ax) / den

	if t <= 0 || t >= 1 || u <= 0 || u >= 1 {
		return nil
	}

	x := a1[0] + t*ax
	y := a1[1] + t*ay

	z := a1[2] + t*(a2[2]-a1[2])

	return &node{
		coord:    [3]float64{x, y, z},
		isInside: true,
	}
}

func pointOnEdge(px, py, x1, y1, x2, y2 float64) bool {
	// Cross product for collinearity
	cross := (x2-x1)*(py-y1) - (y2-y1)*(px-x1)
	if math.Abs(cross) > 1e-9 {
		return false
	}
	// Check if point is within bounding box
	minX, maxX := x1, x2
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := y1, y2
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	return px >= minX && px <= maxX && py >= minY && py <= maxY
}
