package clip

import (
	"math"
)

const eps = 1e-9

type Vec3 struct{ X, Y, Z float64 }
type Vec2 struct{ X, Y float64 }

// Attributes can be expanded to include UV (U, V) or Normals
type Vertex struct {
	Pos  Vec3
	U, V float64
}

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

// Edge represents a unique connection between two vertex indices
type Edge [2]int

func NewEdge(a, b int) Edge {
	if a < b {
		return Edge{a, b}
	}
	return Edge{b, a}
}

// ----------------------------------------------------------------
// Geometry Helpers
// ----------------------------------------------------------------

func toVec2(v Vertex) Vec2 { return Vec2{v.Pos.X, v.Pos.Y} }

func cross2(a, b Vec2) float64 { return a.X*b.Y - a.Y*b.X }
func sub2(a, b Vec2) Vec2      { return Vec2{a.X - b.X, a.Y - b.Y} }

func insideHalfPlane(p Vec2, line [2]Vec2) bool {
	return cross2(sub2(line[1], line[0]), sub2(p, line[0])) >= -eps
}

// intersect interpolates between two vertices based on the 2D clip line
func intersect(v1, v2 Vertex, c1, c2 Vec2) Vertex {
	p1, p2 := toVec2(v1), toVec2(v2)

	r := sub2(p2, p1)
	s := sub2(c2, c1)
	den := cross2(r, s)

	t := 0.0
	if math.Abs(den) > eps {
		t = cross2(sub2(c1, p1), s) / den
	}

	// Clamp to [0, 1] to handle floating point drift
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	// Linear interpolation of Position and UVs
	return Vertex{
		Pos: v1.Pos.Add(v2.Pos.Sub(v1.Pos).Mul(t)),
		U:   v1.U + (v2.U-v1.U)*t,
		V:   v1.V + (v2.V-v1.V)*t,
	}
}

// ----------------------------------------------------------------
// Core Clipping Logic
// ----------------------------------------------------------------

// ClipMesh takes a list of faces (indices) and the global vertex list,
// returning a new list of faces.
func ClipMesh(faces [][]int, vertices *[]Vertex, clipFrame []Vec2) [][]int {
	splitMap := make(map[Edge]int)
	var newFaces [][]int

	for _, face := range faces {
		clippedPoly := clipFace(face, vertices, clipFrame, splitMap)
		if len(clippedPoly) >= 3 {
			// If you need strictly triangles, you can triangulate here
			newFaces = append(newFaces, clippedPoly)
		}
	}
	return newFaces
}

func clipFace(faceIndices []int, vertices *[]Vertex, clipFrame []Vec2, splitMap map[Edge]int) []int {
	output := faceIndices

	for i := range clipFrame {
		if len(output) == 0 {
			return nil
		}

		clipStart := clipFrame[i]
		clipEnd := clipFrame[(i+1)%len(clipFrame)]
		line := [2]Vec2{clipStart, clipEnd}

		var nextOutput []int

		for j := 0; j < len(output); j++ {
			currIdx := output[j]
			prevIdx := output[(j+len(output)-1)%len(output)]

			currV := (*vertices)[currIdx]
			prevV := (*vertices)[prevIdx]

			currIn := insideHalfPlane(toVec2(currV), line)
			prevIn := insideHalfPlane(toVec2(prevV), line)

			if currIn {
				if !prevIn {
					// Entry point: Intersect
					interV := intersect(prevV, currV, clipStart, clipEnd)
					newIdx := getOrUpdateSplit(prevIdx, currIdx, interV, vertices, splitMap)
					nextOutput = append(nextOutput, newIdx)
				}
				nextOutput = append(nextOutput, currIdx)
			} else if prevIn {
				// Exit point: Intersect
				interV := intersect(prevV, currV, clipStart, clipEnd)
				newIdx := getOrUpdateSplit(prevIdx, currIdx, interV, vertices, splitMap)
				nextOutput = append(nextOutput, newIdx)
			}
		}
		output = nextOutput
	}
	return output
}

// getOrUpdateSplit checks if this edge has already been cut to preserve mesh manifoldness
func getOrUpdateSplit(aIdx, bIdx int, newV Vertex, vertices *[]Vertex, splitMap map[Edge]int) int {
	edge := NewEdge(aIdx, bIdx)
	if existingIdx, ok := splitMap[edge]; ok {
		return existingIdx
	}

	newIdx := len(*vertices)
	*vertices = append(*vertices, newV)
	splitMap[edge] = newIdx
	return newIdx
}

// Triangulate turns a multi-sided polygon index list into a triangle list (fan)
func Triangulate(poly []int) [][]int {
	if len(poly) < 3 {
		return nil
	}
	var tris [][]int
	for i := 1; i < len(poly)-1; i++ {
		tris = append(tris, []int{poly[0], poly[i], poly[i+1]})
	}
	return tris
}
