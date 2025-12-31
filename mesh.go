package clippoly

import "fmt"

// ClipMesh clips all faces of a mesh against the provided clip polygon.
// The returned vertices and faces describe the clipped mesh using shared vertices.
func ClipMesh(vertices []Coord, faces [][3]int, clip Polygon) ([]Coord, [][3]int, error) {
	if len(faces) == 0 || len(vertices) == 0 {
		return nil, nil, nil
	}

	vertexIndex := make(map[Coord]int, len(vertices))
	clippedVerts := make([]Coord, 0, len(vertices))
	clippedFaces := make([][3]int, 0, len(faces))

	addVertex := func(v Coord) int {
		if idx, ok := vertexIndex[v]; ok {
			return idx
		}
		idx := len(clippedVerts)
		clippedVerts = append(clippedVerts, v)
		vertexIndex[v] = idx
		return idx
	}

	for _, face := range faces {
		poly := Polygon{
			vertices[face[0]],
			vertices[face[1]],
			vertices[face[2]],
		}

		clipped, err := Clip(poly, clip)
		if err != nil {
			fmt.Println("error: ", err, poly, clip)
			// return nil, nil, err
			continue
		}
		if clipped == nil {
			continue
		}

		for _, tri := range clipped {
			if len(tri) != 3 {
				continue
			}
			var f [3]int
			for i := 0; i < 3; i++ {
				f[i] = addVertex(tri[i])
			}
			clippedFaces = append(clippedFaces, f)
		}
	}

	return clippedVerts, clippedFaces, nil
}
