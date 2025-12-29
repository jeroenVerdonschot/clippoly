package clippoly

import (
	"fmt"
	"math"
)

type coord [3]float32

type Polygon []coord

type Polygons []Polygon

type node struct {
	coord    coord
	isInside bool
	nodes    []*node
	id       int
	isTarget bool
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

func Crop(target, clip Polygon) (triangles Polygons, err error) {
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
		nextNode.remove(curNode)
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

		if intNode := checkIntersections(curNode, n, nodes, idGen); intNode != nil {
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

		if n1.coord == curr.coord || n1.coord == prev.coord || pointOnSegment(px, py, x1, y1, x2, y2) {
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
		coord:    [3]float32{x, y, z},
		isInside: true,
	}
}

func pointOnSegment(px, py, x1, y1, x2, y2 float64) bool {
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
