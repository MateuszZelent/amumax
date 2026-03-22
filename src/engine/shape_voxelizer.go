package engine

import (
	"math"
)

type shapeVoxelizer interface {
	cellMetrics(bounds cellBounds) geomCellMetrics
}

type sampledShapeVoxelizer struct {
	inside func(x, y, z float64) bool
}

type sampledBoundsClass uint8

const (
	sampledBoundsEmpty sampledBoundsClass = iota
	sampledBoundsFull
	sampledBoundsAmbiguous
)

type cellBounds struct {
	xMin float64
	xMax float64
	yMin float64
	yMax float64
	zMin float64
	zMax float64
}

func newDerivedShape(inside func(x, y, z float64) bool, voxelizer shapeVoxelizer) shape {
	if inside == nil {
		return newShape(nil)
	}
	if voxelizer == nil {
		voxelizer = sampledShapeVoxelizer{inside: inside}
	}
	return newVoxelizedShape(inside, voxelizer)
}

func boundsFromIndex(ix, iy, iz int) cellBounds {
	center := index2Coord(ix, iy, iz)
	cell := GetMesh().CellSize()
	return cellBounds{
		xMin: center[X] - cell[X]/2,
		xMax: center[X] + cell[X]/2,
		yMin: center[Y] - cell[Y]/2,
		yMax: center[Y] + cell[Y]/2,
		zMin: center[Z] - cell[Z]/2,
		zMax: center[Z] + cell[Z]/2,
	}
}

func (b cellBounds) translated(dx, dy, dz float64) cellBounds {
	return cellBounds{
		xMin: b.xMin + dx,
		xMax: b.xMax + dx,
		yMin: b.yMin + dy,
		yMax: b.yMax + dy,
		zMin: b.zMin + dz,
		zMax: b.zMax + dz,
	}
}

func (b cellBounds) scaled(sx, sy, sz float64) cellBounds {
	xMin, xMax := scaledBounds(b.xMin, b.xMax, sx)
	yMin, yMax := scaledBounds(b.yMin, b.yMax, sy)
	zMin, zMax := scaledBounds(b.zMin, b.zMax, sz)
	return cellBounds{xMin: xMin, xMax: xMax, yMin: yMin, yMax: yMax, zMin: zMin, zMax: zMax}
}

func scaledBounds(min, max, scale float64) (float64, float64) {
	a := min / scale
	b := max / scale
	if a <= b {
		return a, b
	}
	return b, a
}

func overlap1D(aMin, aMax, bMin, bMax float64) float64 {
	min := math.Max(aMin, bMin)
	max := math.Min(aMax, bMax)
	if max <= min {
		return 0
	}
	return max - min
}

func midpointAverage(a, b float64, fn func(float64) float64) float64 {
	if b <= a {
		return 0
	}
	n := waveguideQuadratureSamples()
	step := (b - a) / float64(n)
	var sum float64
	for i := 0; i < n; i++ {
		x := a + (float64(i)+0.5)*step
		sum += fn(x)
	}
	return sum / float64(n)
}

func adaptiveAverage(a, b float64, fn func(float64) float64) float64 {
	if b <= a {
		return 0
	}
	if GeomMaxDepth <= 0 {
		return midpointAverage(a, b, fn)
	}
	fa := fn(a)
	fm := fn((a + b) / 2)
	fb := fn(b)
	integral := adaptiveSimpson(a, b, fa, fm, fb, 0, fn)
	return integral / (b - a)
}

func adaptiveSimpson(a, b, fa, fm, fb float64, depth int, fn func(float64) float64) float64 {
	whole := simpsonEstimate(a, b, fa, fm, fb)
	mid := (a + b) / 2
	leftMid := (a + mid) / 2
	rightMid := (mid + b) / 2
	flm := fn(leftMid)
	frm := fn(rightMid)
	left := simpsonEstimate(a, mid, fa, flm, fm)
	right := simpsonEstimate(mid, b, fm, frm, fb)
	err := math.Abs(left + right - whole)
	tol := math.Max(GeomTol, 1e-6) * (b - a)
	if depth >= GeomMaxDepth || err <= 15*tol {
		return left + right + (left+right-whole)/15
	}
	return adaptiveSimpson(a, mid, fa, flm, fm, depth+1, fn) + adaptiveSimpson(mid, b, fm, frm, fb, depth+1, fn)
}

func simpsonEstimate(a, b, fa, fm, fb float64) float64 {
	return (b - a) * (fa + 4*fm + fb) / 6
}

func waveguideQuadratureSamples() int {
	n := edgeSmoothSamples()
	if n < 8 {
		return 8
	}
	return n
}

func faceEpsilon(span float64) float64 {
	eps := math.Abs(span) * 1e-12
	if eps == 0 {
		return 1e-12
	}
	return eps
}

func sampledLeafSamples() int {
	n := edgeSmoothSamples()
	if n < 4 {
		return 4
	}
	return n
}

func sampledSnapUnit(v float64) float64 {
	if v <= GeomTol {
		return 0
	}
	if 1-v <= GeomTol {
		return 1
	}
	return v
}

func interiorCoord(min, max, t float64) float64 {
	if max <= min {
		return min
	}
	eps := faceEpsilon(max - min)
	lo := min + eps
	hi := max - eps
	if hi < lo {
		return (min + max) / 2
	}
	return lo + (hi-lo)*t
}

func (b cellBounds) midpoint() (float64, float64, float64) {
	return (b.xMin + b.xMax) / 2, (b.yMin + b.yMax) / 2, (b.zMin + b.zMax) / 2
}

func (b cellBounds) samplePoint(tx, ty, tz float64) (float64, float64, float64) {
	return interiorCoord(b.xMin, b.xMax, tx), interiorCoord(b.yMin, b.yMax, ty), interiorCoord(b.zMin, b.zMax, tz)
}

func (b cellBounds) subdivide() [8]cellBounds {
	xMid, yMid, zMid := b.midpoint()
	return [8]cellBounds{
		{xMin: b.xMin, xMax: xMid, yMin: b.yMin, yMax: yMid, zMin: b.zMin, zMax: zMid},
		{xMin: xMid, xMax: b.xMax, yMin: b.yMin, yMax: yMid, zMin: b.zMin, zMax: zMid},
		{xMin: b.xMin, xMax: xMid, yMin: yMid, yMax: b.yMax, zMin: b.zMin, zMax: zMid},
		{xMin: xMid, xMax: b.xMax, yMin: yMid, yMax: b.yMax, zMin: b.zMin, zMax: zMid},
		{xMin: b.xMin, xMax: xMid, yMin: b.yMin, yMax: yMid, zMin: zMid, zMax: b.zMax},
		{xMin: xMid, xMax: b.xMax, yMin: b.yMin, yMax: yMid, zMin: zMid, zMax: b.zMax},
		{xMin: b.xMin, xMax: xMid, yMin: yMid, yMax: b.yMax, zMin: zMid, zMax: b.zMax},
		{xMin: xMid, xMax: b.xMax, yMin: yMid, yMax: b.yMax, zMin: zMid, zMax: b.zMax},
	}
}

type faceBounds struct {
	axis       int
	coord      float64
	uMin, uMax float64
	vMin, vMax float64
}

func newFaceBounds(bounds cellBounds, axis int, positive bool) faceBounds {
	switch axis {
	case X:
		x := bounds.xMin + faceEpsilon(bounds.xMax-bounds.xMin)
		if positive {
			x = bounds.xMax - faceEpsilon(bounds.xMax-bounds.xMin)
		}
		return faceBounds{axis: axis, coord: x, uMin: bounds.yMin, uMax: bounds.yMax, vMin: bounds.zMin, vMax: bounds.zMax}
	case Y:
		y := bounds.yMin + faceEpsilon(bounds.yMax-bounds.yMin)
		if positive {
			y = bounds.yMax - faceEpsilon(bounds.yMax-bounds.yMin)
		}
		return faceBounds{axis: axis, coord: y, uMin: bounds.xMin, uMax: bounds.xMax, vMin: bounds.zMin, vMax: bounds.zMax}
	case Z:
		z := bounds.zMin + faceEpsilon(bounds.zMax-bounds.zMin)
		if positive {
			z = bounds.zMax - faceEpsilon(bounds.zMax-bounds.zMin)
		}
		return faceBounds{axis: axis, coord: z, uMin: bounds.xMin, uMax: bounds.xMax, vMin: bounds.yMin, vMax: bounds.yMax}
	default:
		return faceBounds{}
	}
}

func (f faceBounds) samplePoint(tu, tv float64) (float64, float64, float64) {
	u := interiorCoord(f.uMin, f.uMax, tu)
	v := interiorCoord(f.vMin, f.vMax, tv)
	switch f.axis {
	case X:
		return f.coord, u, v
	case Y:
		return u, f.coord, v
	case Z:
		return u, v, f.coord
	default:
		return 0, 0, 0
	}
}

func (f faceBounds) subdivide() [4]faceBounds {
	uMid := (f.uMin + f.uMax) / 2
	vMid := (f.vMin + f.vMax) / 2
	return [4]faceBounds{
		{axis: f.axis, coord: f.coord, uMin: f.uMin, uMax: uMid, vMin: f.vMin, vMax: vMid},
		{axis: f.axis, coord: f.coord, uMin: uMid, uMax: f.uMax, vMin: f.vMin, vMax: vMid},
		{axis: f.axis, coord: f.coord, uMin: f.uMin, uMax: uMid, vMin: vMid, vMax: f.vMax},
		{axis: f.axis, coord: f.coord, uMin: uMid, uMax: f.uMax, vMin: vMid, vMax: f.vMax},
	}
}

func (v sampledShapeVoxelizer) cellMetrics(bounds cellBounds) geomCellMetrics {
	var faces [6]float32
	faces[0] = float32(v.faceFraction(newFaceBounds(bounds, X, false), 0))
	faces[1] = float32(v.faceFraction(newFaceBounds(bounds, X, true), 0))
	faces[2] = float32(v.faceFraction(newFaceBounds(bounds, Y, false), 0))
	faces[3] = float32(v.faceFraction(newFaceBounds(bounds, Y, true), 0))
	faces[4] = float32(v.faceFraction(newFaceBounds(bounds, Z, false), 0))
	faces[5] = float32(v.faceFraction(newFaceBounds(bounds, Z, true), 0))
	return newGeomCellMetrics(float32(v.volumeFraction(bounds, 0)), faces)
}

func (v sampledShapeVoxelizer) volumeFraction(bounds cellBounds, depth int) float64 {
	switch v.classifyBounds(bounds) {
	case sampledBoundsEmpty:
		return 0
	case sampledBoundsFull:
		return 1
	}
	if depth >= GeomMaxDepth {
		return sampledSnapUnit(v.sampleVolume(bounds))
	}
	children := bounds.subdivide()
	var sum float64
	for _, child := range children {
		sum += v.volumeFraction(child, depth+1)
	}
	return sampledSnapUnit(sum / float64(len(children)))
}

func (v sampledShapeVoxelizer) faceFraction(face faceBounds, depth int) float64 {
	switch v.classifyFace(face) {
	case sampledBoundsEmpty:
		return 0
	case sampledBoundsFull:
		return 1
	}
	if depth >= GeomMaxDepth {
		return sampledSnapUnit(v.sampleFace(face))
	}
	children := face.subdivide()
	var sum float64
	for _, child := range children {
		sum += v.faceFraction(child, depth+1)
	}
	return sampledSnapUnit(sum / float64(len(children)))
}

func (v sampledShapeVoxelizer) classifyBounds(bounds cellBounds) sampledBoundsClass {
	samples := [][3]float64{
		{0, 0, 0},
		{1, 0, 0},
		{0, 1, 0},
		{1, 1, 0},
		{0, 0, 1},
		{1, 0, 1},
		{0, 1, 1},
		{1, 1, 1},
		{0.5, 0.5, 0.5},
	}
	var insideCount int
	for _, sample := range samples {
		x, y, z := bounds.samplePoint(sample[0], sample[1], sample[2])
		if v.inside(x, y, z) {
			insideCount++
		}
	}
	switch {
	case insideCount == 0:
		return sampledBoundsEmpty
	case insideCount == len(samples):
		return sampledBoundsFull
	default:
		return sampledBoundsAmbiguous
	}
}

func (v sampledShapeVoxelizer) classifyFace(face faceBounds) sampledBoundsClass {
	samples := [][2]float64{
		{0, 0},
		{1, 0},
		{0, 1},
		{1, 1},
		{0.5, 0.5},
	}
	var insideCount int
	for _, sample := range samples {
		x, y, z := face.samplePoint(sample[0], sample[1])
		if v.inside(x, y, z) {
			insideCount++
		}
	}
	switch {
	case insideCount == 0:
		return sampledBoundsEmpty
	case insideCount == len(samples):
		return sampledBoundsFull
	default:
		return sampledBoundsAmbiguous
	}
}

func (v sampledShapeVoxelizer) sampleVolume(bounds cellBounds) float64 {
	n := sampledLeafSamples()
	var insideCount int
	for ix := 0; ix < n; ix++ {
		tx := (float64(ix) + 0.5) / float64(n)
		for iy := 0; iy < n; iy++ {
			ty := (float64(iy) + 0.5) / float64(n)
			for iz := 0; iz < n; iz++ {
				tz := (float64(iz) + 0.5) / float64(n)
				x, y, z := bounds.samplePoint(tx, ty, tz)
				if v.inside(x, y, z) {
					insideCount++
				}
			}
		}
	}
	return float64(insideCount) / float64(n*n*n)
}

func (v sampledShapeVoxelizer) sampleFace(face faceBounds) float64 {
	n := sampledLeafSamples()
	var insideCount int
	for iu := 0; iu < n; iu++ {
		tu := (float64(iu) + 0.5) / float64(n)
		for iv := 0; iv < n; iv++ {
			tv := (float64(iv) + 0.5) / float64(n)
			x, y, z := face.samplePoint(tu, tv)
			if v.inside(x, y, z) {
				insideCount++
			}
		}
	}
	return float64(insideCount) / float64(n*n)
}

type translatedShapeVoxelizer struct {
	base       shapeVoxelizer
	dx, dy, dz float64
}

func (v translatedShapeVoxelizer) cellMetrics(bounds cellBounds) geomCellMetrics {
	return v.base.cellMetrics(bounds.translated(-v.dx, -v.dy, -v.dz))
}

type scaledShapeVoxelizer struct {
	base       shapeVoxelizer
	sx, sy, sz float64
}

func (v scaledShapeVoxelizer) cellMetrics(bounds cellBounds) geomCellMetrics {
	return v.base.cellMetrics(bounds.scaled(v.sx, v.sy, v.sz))
}

type waveguideIntervalFunc func(x float64) (zMin, zMax float64, ok bool)

type waveguideVoxelizer struct {
	yMin      float64
	yMax      float64
	zInterval waveguideIntervalFunc
}

func (v waveguideVoxelizer) cellMetrics(bounds cellBounds) geomCellMetrics {
	var faces [6]float32
	faces[0] = v.faceFraction(bounds, X, false)
	faces[1] = v.faceFraction(bounds, X, true)
	faces[2] = v.faceFraction(bounds, Y, false)
	faces[3] = v.faceFraction(bounds, Y, true)
	faces[4] = v.faceFraction(bounds, Z, false)
	faces[5] = v.faceFraction(bounds, Z, true)

	yFrac := overlap1D(bounds.yMin, bounds.yMax, v.yMin, v.yMax) / (bounds.yMax - bounds.yMin)
	if yFrac == 0 {
		return newGeomCellMetrics(0, faces)
	}
	xzFrac := adaptiveAverage(bounds.xMin, bounds.xMax, func(x float64) float64 {
		return v.zOverlapFractionAtX(x, bounds.zMin, bounds.zMax)
	})
	return newGeomCellMetrics(float32(yFrac*xzFrac), faces)
}

func (v waveguideVoxelizer) faceFraction(bounds cellBounds, axis int, positive bool) float32 {
	switch axis {
	case X:
		yFrac := overlap1D(bounds.yMin, bounds.yMax, v.yMin, v.yMax) / (bounds.yMax - bounds.yMin)
		if yFrac == 0 {
			return 0
		}
		x := bounds.xMin + faceEpsilon(bounds.xMax-bounds.xMin)
		if positive {
			x = bounds.xMax - faceEpsilon(bounds.xMax-bounds.xMin)
		}
		return float32(yFrac * v.zOverlapFractionAtX(x, bounds.zMin, bounds.zMax))
	case Y:
		y := bounds.yMin + faceEpsilon(bounds.yMax-bounds.yMin)
		if positive {
			y = bounds.yMax - faceEpsilon(bounds.yMax-bounds.yMin)
		}
		if y < v.yMin || y > v.yMax {
			return 0
		}
		xzFrac := adaptiveAverage(bounds.xMin, bounds.xMax, func(x float64) float64 {
			return v.zOverlapFractionAtX(x, bounds.zMin, bounds.zMax)
		})
		return float32(xzFrac)
	case Z:
		yFrac := overlap1D(bounds.yMin, bounds.yMax, v.yMin, v.yMax) / (bounds.yMax - bounds.yMin)
		if yFrac == 0 {
			return 0
		}
		z := bounds.zMin + faceEpsilon(bounds.zMax-bounds.zMin)
		if positive {
			z = bounds.zMax - faceEpsilon(bounds.zMax-bounds.zMin)
		}
		xFrac := adaptiveAverage(bounds.xMin, bounds.xMax, func(x float64) float64 {
			zMin, zMax, ok := v.zInterval(x)
			if !ok {
				return 0
			}
			if z >= zMin && z <= zMax {
				return 1
			}
			return 0
		})
		return float32(yFrac * xFrac)
	default:
		return 0
	}
}

func (v waveguideVoxelizer) zOverlapFractionAtX(x, zMin, zMax float64) float64 {
	lower, upper, ok := v.zInterval(x)
	if !ok {
		return 0
	}
	return overlap1D(zMin, zMax, lower, upper) / (zMax - zMin)
}

type normalWaveguideVoxelizer struct {
	halfL         float64
	halfH         float64
	centerZ       func(float64) float64
	dCenterZ      func(float64) float64
	ddCenterZ     func(float64) float64
	projectionTol float64
}

func (v normalWaveguideVoxelizer) zInterval(x float64) (float64, float64, bool) {
	if x < -v.halfL || x > v.halfL {
		return 0, 0, false
	}

	lower := math.Inf(1)
	upper := math.Inf(-1)

	if u, ok := v.solveOffsetParameter(x, false); ok {
		z := v.offsetZ(u, false)
		if z < lower {
			lower = z
		}
		if z > upper {
			upper = z
		}
	}
	if u, ok := v.solveOffsetParameter(x, true); ok {
		z := v.offsetZ(u, true)
		if z < lower {
			lower = z
		}
		if z > upper {
			upper = z
		}
	}

	if dx := x + v.halfL; dx <= v.halfH {
		dz := math.Sqrt(math.Max(0, v.halfH*v.halfH-dx*dx))
		z0 := v.centerZ(-v.halfL)
		if z0-dz < lower {
			lower = z0 - dz
		}
		if z0+dz > upper {
			upper = z0 + dz
		}
	}
	if dx := v.halfL - x; dx <= v.halfH {
		dz := math.Sqrt(math.Max(0, v.halfH*v.halfH-dx*dx))
		z0 := v.centerZ(v.halfL)
		if z0-dz < lower {
			lower = z0 - dz
		}
		if z0+dz > upper {
			upper = z0 + dz
		}
	}

	if lower > upper {
		return 0, 0, false
	}
	return lower, upper, true
}

func (v normalWaveguideVoxelizer) solveOffsetParameter(x float64, upper bool) (float64, bool) {
	dir := -1.0
	if upper {
		dir = 1.0
	}
	u := clampFloat64(x, -v.halfL, v.halfL)
	for iter := 0; iter < 12; iter++ {
		slope := v.dCenterZ(u)
		norm := math.Sqrt(1 + slope*slope)
		curvatureTerm := v.ddCenterZ(u) / (norm * norm * norm)
		f := u - dir*v.halfH*slope/norm - x
		df := 1 - dir*v.halfH*curvatureTerm
		if math.Abs(df) < 1e-18 {
			break
		}
		next := clampFloat64(u-f/df, -v.halfL, v.halfL)
		if math.Abs(next-u) <= v.projectionTol {
			u = next
			break
		}
		u = next
	}
	slope := v.dCenterZ(u)
	norm := math.Sqrt(1 + slope*slope)
	residual := math.Abs(u - dir*v.halfH*slope/norm - x)
	limit := math.Max(v.projectionTol*10, v.halfH*1e-6)
	if limit == 0 {
		limit = 1e-12
	}
	return u, residual <= limit
}

func (v normalWaveguideVoxelizer) offsetZ(u float64, upper bool) float64 {
	dir := -1.0
	if upper {
		dir = 1.0
	}
	slope := v.dCenterZ(u)
	norm := math.Sqrt(1 + slope*slope)
	return v.centerZ(u) + dir*v.halfH/norm
}

func newVerticalWaveguideShape(length, width, height float64, centerZ func(float64) float64) shape {
	halfL := length / 2
	halfW := width / 2
	halfH := height / 2
	inside := func(x, y, z float64) bool {
		if x < -halfL || x > halfL || y < -halfW || y > halfW {
			return false
		}
		zCenter := centerZ(x)
		return z >= zCenter-halfH && z < zCenter+halfH
	}
	voxelizer := waveguideVoxelizer{
		yMin: -halfW,
		yMax: halfW,
		zInterval: func(x float64) (float64, float64, bool) {
			if x < -halfL || x > halfL {
				return 0, 0, false
			}
			zCenter := centerZ(x)
			return zCenter - halfH, zCenter + halfH, true
		},
	}
	return newVoxelizedShape(inside, voxelizer)
}

func newNormalWaveguideShape(length, width, height float64, centerZ, dCenterZ, ddCenterZ func(float64) float64) shape {
	halfL := length / 2
	halfW := width / 2
	halfH := height / 2
	projectionTol := math.Max(length, height) * 1e-12
	if projectionTol == 0 {
		projectionTol = 1e-12
	}

	inside := func(x, y, z float64) bool {
		if x < -halfL || x > halfL || y < -halfW || y > halfW {
			return false
		}

		u := clampFloat64(x, -halfL, halfL)
		for iter := 0; iter < 8; iter++ {
			zu := centerZ(u)
			slope := dCenterZ(u)
			f := (u - x) + (zu-z)*slope
			df := 1 + slope*slope + (zu-z)*ddCenterZ(u)
			if math.Abs(df) < 1e-18 {
				break
			}
			next := clampFloat64(u-f/df, -halfL, halfL)
			if math.Abs(next-u) <= projectionTol {
				u = next
				break
			}
			u = next
		}

		dx := x - u
		dz := z - centerZ(u)
		return dx*dx+dz*dz <= halfH*halfH
	}
	voxelizer := waveguideVoxelizer{
		yMin: -halfW,
		yMax: halfW,
		zInterval: normalWaveguideVoxelizer{
			halfL:         halfL,
			halfH:         halfH,
			centerZ:       centerZ,
			dCenterZ:      dCenterZ,
			ddCenterZ:     ddCenterZ,
			projectionTol: projectionTol,
		}.zInterval,
	}
	return newVoxelizedShape(inside, voxelizer)
}
