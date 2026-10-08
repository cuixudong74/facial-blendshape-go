package blendshape

import "math"

// makeMesh 构造 n 顶点基础网格，顶点 i 位于 (i, i, i)。
func makeMesh(n int) *Mesh {
	pos := make([]float32, 3*n)
	for i := 0; i < n; i++ {
		pos[3*i] = float32(i)
		pos[3*i+1] = float32(i)
		pos[3*i+2] = float32(i)
	}
	return &Mesh{Positions: pos, VertexCount: n}
}

// makeTarget 构造所有顶点增量均为 (dx,dy,dz) 的目标。
func makeTarget(name string, layer Layer, n int, dx, dy, dz float32) *MorphTarget {
	d := make([]float32, 3*n)
	for i := 0; i < n; i++ {
		d[3*i] = dx
		d[3*i+1] = dy
		d[3*i+2] = dz
	}
	return &MorphTarget{Name: name, Layer: layer, Deltas: d, VertexCount: n}
}

func close32(a, b float32) bool {
	return math.Abs(float64(a-b)) < 1e-5
}

func close64(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
