package engine

import (
	"math"
	"sort"
)

const (
	guideProjectionIterations = 8
	guideMinLUTSamples        = 512
	guideMaxLUTSamples        = 16384
)

type planarGuideGeometry struct {
	length float64
	width  float64
	height float64
	halfL  float64
	halfW  float64
	halfH  float64

	centerZ   func(float64) float64
	dCenterZ  func(float64) float64
	ddCenterZ func(float64) float64

	xs   []float64
	ss   []float64
	sMax float64
	zMin float64
	zMax float64
}

type sinGuideGeometry struct {
	planarGuideGeometry
	period float64
	amp    float64
	phase  float64
	z0     float64
}

type archGuideGeometry struct {
	planarGuideGeometry
	archHeight float64
	z0         float64
}

func newSinGuideGeometry(length, width, height, period, centerAmp, phase, z0 float64) *sinGuideGeometry {
	k := 2 * math.Pi / period
	g := &sinGuideGeometry{
		period: period,
		amp:    centerAmp,
		phase:  phase,
		z0:     z0,
	}
	g.planarGuideGeometry = newPlanarGuideGeometry(
		length,
		width,
		height,
		func(x float64) float64 {
			return z0 + centerAmp*math.Sin(k*x+phase)
		},
		func(x float64) float64 {
			return centerAmp * k * math.Cos(k*x+phase)
		},
		func(x float64) float64 {
			return -centerAmp * k * k * math.Sin(k*x+phase)
		},
		clampGuideLUTSamples(int(math.Ceil(math.Max(1, length/period)*256))),
	)
	return g
}

func newArchGuideGeometry(length, width, height, archHeight, z0 float64) *archGuideGeometry {
	halfL := length / 2
	g := &archGuideGeometry{
		archHeight: archHeight,
		z0:         z0,
	}
	g.planarGuideGeometry = newPlanarGuideGeometry(
		length,
		width,
		height,
		func(x float64) float64 {
			t := (x + halfL) / length
			return z0 + archHeight*math.Sin(math.Pi*t)
		},
		func(x float64) float64 {
			t := (x + halfL) / length
			return archHeight * (math.Pi / length) * math.Cos(math.Pi*t)
		},
		func(x float64) float64 {
			t := (x + halfL) / length
			scale := math.Pi / length
			return -archHeight * scale * scale * math.Sin(math.Pi*t)
		},
		2048,
	)
	return g
}

func newPlanarGuideGeometry(length, width, height float64, centerZ, dCenterZ, ddCenterZ func(float64) float64, lutSamples int) planarGuideGeometry {
	g := planarGuideGeometry{
		length:    length,
		width:     width,
		height:    height,
		halfL:     length / 2,
		halfW:     width / 2,
		halfH:     height / 2,
		centerZ:   centerZ,
		dCenterZ:  dCenterZ,
		ddCenterZ: ddCenterZ,
	}
	g.buildArcLUT(clampGuideLUTSamples(lutSamples))
	return g
}

func clampGuideLUTSamples(n int) int {
	if n < guideMinLUTSamples {
		return guideMinLUTSamples
	}
	if n > guideMaxLUTSamples {
		return guideMaxLUTSamples
	}
	return n
}

func (g *planarGuideGeometry) buildArcLUT(n int) {
	g.xs = make([]float64, n+1)
	g.ss = make([]float64, n+1)
	step := g.length / float64(n)
	g.zMin = math.Inf(1)
	g.zMax = math.Inf(-1)

	prevX := -g.halfL
	prevSpeed := g.arcSpeed(prevX)
	for i := 0; i <= n; i++ {
		x := -g.halfL + float64(i)*step
		if i == n {
			x = g.halfL
		}
		g.xs[i] = x
		z := g.centerZ(x)
		if z < g.zMin {
			g.zMin = z
		}
		if z > g.zMax {
			g.zMax = z
		}
		if i == 0 {
			continue
		}
		speed := g.arcSpeed(x)
		g.ss[i] = g.ss[i-1] + 0.5*(prevSpeed+speed)*(x-prevX)
		prevX = x
		prevSpeed = speed
	}
	g.sMax = g.ss[len(g.ss)-1]
	g.zMin -= g.halfH
	g.zMax += g.halfH
}

func (g *planarGuideGeometry) SRange() (float64, float64) {
	return 0, g.sMax
}

func (g *planarGuideGeometry) FrameAtS(s float64) guideFrame {
	x := g.xFromS(s)
	slope := g.dCenterZ(x)
	invNorm := 1 / math.Sqrt(1+slope*slope)
	return guideFrame{
		R: vec3{X: x, Y: 0, Z: g.centerZ(x)},
		T: vec3{X: invNorm, Y: 0, Z: slope * invNorm},
		V: vec3{X: 0, Y: 1, Z: 0},
		W: vec3{X: -slope * invNorm, Y: 0, Z: invNorm},
	}
}

func (g *planarGuideGeometry) ProjectPoint(x, y, z float64) (s, v, w float64, ok bool) {
	projectedX, ok := g.projectX(x, z)
	if !ok {
		return 0, 0, 0, false
	}
	s = g.sFromX(projectedX)
	frame := g.FrameAtS(s)
	delta := vec3{X: x, Y: y, Z: z}.sub(frame.R)
	return s, delta.dot(frame.V), delta.dot(frame.W), true
}

func (g *planarGuideGeometry) CrossSectionArea(s float64) float64 {
	_ = s
	return g.width * g.height
}

func (g *planarGuideGeometry) CrossSectionBounds(s float64) (vMin, vMax, wMin, wMax float64) {
	_ = s
	return -g.halfW, g.halfW, -g.halfH, g.halfH
}

func (g *planarGuideGeometry) BoundingBox() (xmin, xmax, ymin, ymax, zmin, zmax float64) {
	return -g.halfL, g.halfL, -g.halfW, g.halfW, g.zMin, g.zMax
}

func (g *planarGuideGeometry) arcSpeed(x float64) float64 {
	slope := g.dCenterZ(x)
	return math.Sqrt(1 + slope*slope)
}

func (g *planarGuideGeometry) sFromX(x float64) float64 {
	x = clampFloat64(x, -g.halfL, g.halfL)
	i := sort.Search(len(g.xs), func(i int) bool { return g.xs[i] >= x })
	if i <= 0 {
		return 0
	}
	if i >= len(g.xs) {
		return g.sMax
	}
	x0, x1 := g.xs[i-1], g.xs[i]
	if x1 <= x0 {
		return g.ss[i]
	}
	t := (x - x0) / (x1 - x0)
	return g.ss[i-1] + t*(g.ss[i]-g.ss[i-1])
}

func (g *planarGuideGeometry) xFromS(s float64) float64 {
	s = clampFloat64(s, 0, g.sMax)
	i := sort.Search(len(g.ss), func(i int) bool { return g.ss[i] >= s })
	if i <= 0 {
		return -g.halfL
	}
	if i >= len(g.ss) {
		return g.halfL
	}
	s0, s1 := g.ss[i-1], g.ss[i]
	if s1 <= s0 {
		return g.xs[i]
	}
	t := (s - s0) / (s1 - s0)
	return g.xs[i-1] + t*(g.xs[i]-g.xs[i-1])
}

func (g *planarGuideGeometry) projectX(px, pz float64) (float64, bool) {
	x := clampFloat64(px, -g.halfL, g.halfL)
	for iter := 0; iter < guideProjectionIterations; iter++ {
		z := g.centerZ(x)
		slope := g.dCenterZ(x)
		curvature := g.ddCenterZ(x)
		f := (x - px) + (z-pz)*slope
		df := 1 + slope*slope + (z-pz)*curvature
		if math.Abs(df) < 1e-18 {
			break
		}
		next := clampFloat64(x-f/df, -g.halfL, g.halfL)
		if math.Abs(next-x) <= g.projectionTolerance() {
			x = next
			break
		}
		x = next
	}

	bestX := x
	bestDist2 := g.distanceSquaredToCenterline(x, px, pz)
	for _, candidate := range []float64{-g.halfL, g.halfL} {
		dist2 := g.distanceSquaredToCenterline(candidate, px, pz)
		if dist2 < bestDist2 {
			bestX = candidate
			bestDist2 = dist2
		}
	}
	return bestX, true
}

func (g *planarGuideGeometry) distanceSquaredToCenterline(x, px, pz float64) float64 {
	dx := x - px
	dz := g.centerZ(x) - pz
	return dx*dx + dz*dz
}

func (g *planarGuideGeometry) projectionTolerance() float64 {
	tol := math.Max(g.length, g.height) * 1e-12
	if tol == 0 {
		return 1e-12
	}
	return tol
}
