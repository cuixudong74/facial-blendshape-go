package blendshape

// Vec3 is a 3-component vector. float64 is used deliberately so that long
// running animation sampling accumulates no float32 rounding jitter.
type Vec3 struct {
	X, Y, Z float64
}

// Add returns a+b.
func Add(a, b Vec3) Vec3 {
	return Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

// Scale returns v*s.
func Scale(v Vec3, s float64) Vec3 {
	return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

// Lerp returns a+(b-a)*t. float64 arithmetic makes repeated sampling
// reproducible: the same (a,b,t) always yields identical bits.
func Lerp(a, b Vec3, t float64) Vec3 {
	return Vec3{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
}
