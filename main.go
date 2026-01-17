package clippoly

import (
	"fmt"
	"math"
)

const eps = 1e-9

type Vec3 struct{ X, Y, Z float64 }
type Vec2 struct{ X, Y float64 }

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

func cross2(a, b Vec2) float64 { return a.X*b.Y - a.Y*b.X }
func sub2(a, b Vec2) Vec2      { return Vec2{a.X - b.X, a.Y - b.Y} }

func toVec2(p Vec3) Vec2 { return Vec2{p.X, p.Y} }

func sliceFace(face []Vec3, line []Vec2) ([]Vec3, error) {

	for _, vec3 := range face {

		vert2 := toVec2(vec3)
		isInside := insideHalfPlane(vert2, line)

		fmt.Printf("vec3: %v - %v\n", vec3, isInside)

	}

	return nil, nil
}

type Polygon2D []Vec2

func (p Polygon2D) next(idx int) Vec2 {
	next := idx + 1
	if next >= len(p) {
		next = 0
	}
	return p[next]
}

type Polygon3D []Vec3

func (p Polygon3D) next(idx int) Vec3 {
	next := idx + 1
	if next >= len(p) {
		next = 0
	}
	return p[next]
}

func createIDLoop(face Polygon3D) loop {

	loop := make(loop, 3)

	for i := range face {

		next := i + 1

		if next > len(face)-1 {
			next = 0
		}
		loop[i] = []int{i, next}
	}

	return loop
}

type links map[int][]int

func createLinks(face Polygon3D) links {

	links := make(links, len(face))

	for i := range face {

		next := i + 1

		if next > len(face)-1 {
			next = 0
		}

		links[i] = append(links[i], next)
		links[next] = append(links[next], i)
	}

	return links

}

type vertTable map[int]Vec3

func createVertTable(face Polygon3D) vertTable {
	v := make(vertTable, len(face))
	for i, f := range face {
		v[i] = f
	}
	return v
}

func clip(face Polygon3D, clipFrame Polygon2D) {

	loop := createIDLoop(face)
	vertTable := createVertTable(face)

	newIdx := len(face)

	fmt.Println(loop)

	for i := range clipFrame {

		line := []Vec2{clipFrame[i], clipFrame.next(i)}

		for _, e := range loop {

			from, to := vertTable[e[0]], vertTable[e[1]]
			edge := []Vec3{from, to}

			// check if in or out
			isInsideCurr := insideHalfPlane(toVec2(from), line)
			isInsideNext := insideHalfPlane(toVec2(to), line)

			if isInsideCurr && isInsideNext {
				continue
			}

			if !isInsideCurr && !isInsideNext {
				// remove edge
				continue
			}

			// must intersect
			newVec3 := intersect(edge, line)
			// error is not

			_ = newVec3 // add to map

			if !isInsideCurr {
				e[0] = newIdx
			} else {
				e[1] = newIdx
			}

			newIdx++

		}
		fmt.Printf("loop: %v- from clip %v\n", loop, i)
		// fix loop
		loop.fix()
	}
	fmt.Printf("loop: %v\n", loop)
}

type loop [][]int

func (l loop) next(index int) []int {

	next := index + 1
	if next >= len(l) {
		next = 0
	}
	return l[next]

}

func (l *loop) insert(edge []int, index int) {
	if l == nil {
		return
	}

	loop := *l
	if index < 0 {
		index = 0
	}
	if index >= len(loop) {
		loop = append(loop, edge)
		*l = loop
		return
	}

	loop = append(loop, nil)
	copy(loop[index+1:], loop[index:])
	loop[index] = edge
	*l = loop
}

func (l *loop) fix() {

	for idx, edge := range *l {

		if edge[1] != l.next(idx)[0] {
			l.insert([]int{edge[1], l.next(idx)[0]}, idx+1)
		}
	}
}

func lineIntersectionParam(P, Q, A, B Vec2) (t float64, ok bool) {
	r := sub2(Q, P)
	s := sub2(B, A)
	den := cross2(r, s)
	if math.Abs(den) < eps {
		return 0, false
	}
	t = cross2(sub2(A, P), s) / den
	return t, true
}

// if intersects return the inside edge
func intersect(edge []Vec3, cut []Vec2) Vec3 {
	e1, e2 := toVec2(edge[0]), toVec2(edge[1])
	c1, c2 := cut[0], cut[1]

	t, ok := lineIntersectionParam(e1, e2, c1, c2)
	if ok {
		// clamp t to [0,1] to be safe with numeric drift
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		I3 := edge[0].Add(edge[1].Sub(edge[0]).Mul(t))
		return I3
	}

	return Vec3{}
}

func insideHalfPlane(p Vec2, line []Vec2) bool {
	return cross2(sub2(line[1], line[0]), sub2(p, line[0])) >= -eps
}
