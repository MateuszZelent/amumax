package engine

type vec3 struct {
	X float64
	Y float64
	Z float64
}

func (a vec3) add(b vec3) vec3 {
	return vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

func (a vec3) sub(b vec3) vec3 {
	return vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func (a vec3) scale(f float64) vec3 {
	return vec3{X: a.X * f, Y: a.Y * f, Z: a.Z * f}
}

func (a vec3) dot(b vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

type guideFrame struct {
	R vec3
	T vec3
	V vec3
	W vec3
}

type guideGeometry interface {
	SRange() (s0, s1 float64)
	FrameAtS(s float64) guideFrame
	ProjectPoint(x, y, z float64) (s, v, w float64, ok bool)
	CrossSectionArea(s float64) float64
	CrossSectionBounds(s float64) (vMin, vMax, wMin, wMax float64)
	BoundingBox() (xmin, xmax, ymin, ymax, zmin, zmax float64)
}

type translatedGuideGeometry struct {
	base guideGeometry
	dx   float64
	dy   float64
	dz   float64
}

func (g translatedGuideGeometry) SRange() (float64, float64) {
	return g.base.SRange()
}

func (g translatedGuideGeometry) FrameAtS(s float64) guideFrame {
	frame := g.base.FrameAtS(s)
	frame.R = frame.R.add(vec3{X: g.dx, Y: g.dy, Z: g.dz})
	return frame
}

func (g translatedGuideGeometry) ProjectPoint(x, y, z float64) (s, v, w float64, ok bool) {
	return g.base.ProjectPoint(x-g.dx, y-g.dy, z-g.dz)
}

func (g translatedGuideGeometry) CrossSectionArea(s float64) float64 {
	return g.base.CrossSectionArea(s)
}

func (g translatedGuideGeometry) CrossSectionBounds(s float64) (vMin, vMax, wMin, wMax float64) {
	return g.base.CrossSectionBounds(s)
}

func (g translatedGuideGeometry) BoundingBox() (xmin, xmax, ymin, ymax, zmin, zmax float64) {
	xmin, xmax, ymin, ymax, zmin, zmax = g.base.BoundingBox()
	return xmin + g.dx, xmax + g.dx, ymin + g.dy, ymax + g.dy, zmin + g.dz, zmax + g.dz
}
