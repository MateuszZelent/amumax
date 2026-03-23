package engine

import (
	"image"
	_ "image/jpeg" // register JPEG format for image decoding
	_ "image/png"  // register PNG format for image decoding
	"math"

	"github.com/MathieuMoalic/amumax/src/fsutil"
	"github.com/MathieuMoalic/amumax/src/log"
)

// geometrical shape for setting sample geometry
type shape struct {
	insideFn  func(x, y, z float64) bool
	voxelizer shapeVoxelizer
	guide     guideGeometry
}

func newShape(inside func(x, y, z float64) bool) shape {
	return shape{insideFn: inside}
}

func newSampledShape(inside func(x, y, z float64) bool) shape {
	return newDerivedShape(inside, nil)
}

func newVoxelizedShape(inside func(x, y, z float64) bool, voxelizer shapeVoxelizer) shape {
	return shape{insideFn: inside, voxelizer: voxelizer}
}

func newGuideShape(inside func(x, y, z float64) bool, voxelizer shapeVoxelizer, guide guideGeometry) shape {
	if inside == nil {
		return newShape(nil)
	}
	if voxelizer == nil {
		voxelizer = sampledShapeVoxelizer{inside: inside}
	}
	return shape{insideFn: inside, voxelizer: voxelizer, guide: guide}
}

func (s shape) isNil() bool { return s.insideFn == nil }

func (s shape) contains(x, y, z float64) bool {
	return s.insideFn != nil && s.insideFn(x, y, z)
}

// wave with given diameters
func wave(period, amin, amax float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		wavex := (math.Cos(x/period*2*math.Pi)/2 - 0.5) * (amax - amin) / 2
		return y > wavex-amin/2 && y < -wavex+amin/2
	})
}

// sinWaveguide creates a finite sinusoidal waveguide with vertically measured thickness.
// It is a legacy convenience wrapper for SinWaveguideVertical with phase=0 and z0=0.
func sinWaveguide(length, width, height, period, sinAmp float64) shape {
	return sinWaveguideVertical(length, width, height, period, sinAmp, 0, 0)
}

// sinWaveguideVertical creates a finite waveguide whose centerline follows a sinusoidal path,
// with thickness measured vertically along z.
func sinWaveguideVertical(length, width, height, period, centerAmp, phase, z0 float64) shape {
	switch {
	case length <= 0:
		log.Log.ErrAndExit("SinWaveguideVertical: length must be > 0, got %g", length)
	case width <= 0:
		log.Log.ErrAndExit("SinWaveguideVertical: width must be > 0, got %g", width)
	case height <= 0:
		log.Log.ErrAndExit("SinWaveguideVertical: height must be > 0, got %g", height)
	case period <= 0:
		log.Log.ErrAndExit("SinWaveguideVertical: period must be > 0, got %g", period)
	}

	k := 2 * math.Pi / period
	centerZ := func(x float64) float64 {
		return z0 + centerAmp*math.Sin(k*x+phase)
	}
	return newVerticalWaveguideShape(length, width, height, centerZ)
}

// sinWaveguide2 is a backwards-compatible alias for sinWaveguideVertical.
func sinWaveguide2(length, width, height, period, centerAmp, phase, z0 float64) shape {
	return sinWaveguideVertical(length, width, height, period, centerAmp, phase, z0)
}

// archWaveguide creates a half-sine arch waveguide with vertically measured thickness.
// It is a legacy convenience wrapper for ArchWaveguideVertical.
func archWaveguide(length, width, height, archHeight, z0 float64) shape {
	return archWaveguideVertical(length, width, height, archHeight, z0)
}

// archWaveguideVertical creates a waveguide whose centerline follows a half-sine arch,
// with thickness measured vertically along z.
func archWaveguideVertical(length, width, height, archHeight, z0 float64) shape {
	switch {
	case length <= 0:
		log.Log.ErrAndExit("ArchWaveguideVertical: length must be > 0, got %g", length)
	case width <= 0:
		log.Log.ErrAndExit("ArchWaveguideVertical: width must be > 0, got %g", width)
	case height <= 0:
		log.Log.ErrAndExit("ArchWaveguideVertical: height must be > 0, got %g", height)
	}

	halfL := length / 2
	centerZ := func(x float64) float64 {
		t := (x + halfL) / length
		return z0 + archHeight*math.Sin(math.Pi*t)
	}
	return newVerticalWaveguideShape(length, width, height, centerZ)
}

func clampFloat64(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// normalThicknessWaveguide creates a finite waveguide with thickness measured orthogonally
// to the x-z centerline instead of vertically along z.
func normalThicknessWaveguide(length, width, height float64, centerZ, dCenterZ, ddCenterZ func(float64) float64) shape {
	return newNormalWaveguideShape(length, width, height, centerZ, dCenterZ, ddCenterZ)
}

// sinWaveguideNormal creates a sinusoidal waveguide with thickness measured orthogonally
// to the centerline in the x-z plane.
func sinWaveguideNormal(length, width, height, period, centerAmp, phase, z0 float64) shape {
	switch {
	case length <= 0:
		log.Log.ErrAndExit("SinWaveguideNormal: length must be > 0, got %g", length)
	case width <= 0:
		log.Log.ErrAndExit("SinWaveguideNormal: width must be > 0, got %g", width)
	case height <= 0:
		log.Log.ErrAndExit("SinWaveguideNormal: height must be > 0, got %g", height)
	case period <= 0:
		log.Log.ErrAndExit("SinWaveguideNormal: period must be > 0, got %g", period)
	}

	k := 2 * math.Pi / period
	centerZFn := func(x float64) float64 {
		return z0 + centerAmp*math.Sin(k*x+phase)
	}
	dCenterZFn := func(x float64) float64 {
		return centerAmp * k * math.Cos(k*x+phase)
	}
	ddCenterZFn := func(x float64) float64 {
		return -centerAmp * k * k * math.Sin(k*x+phase)
	}

	base := normalThicknessWaveguide(length, width, height, centerZFn, dCenterZFn, ddCenterZFn)
	return newGuideShape(base.insideFn, base.voxelizer, newSinGuideGeometry(length, width, height, period, centerAmp, phase, z0))
}

// archWaveguideNormal creates a half-sine arch waveguide with thickness measured orthogonally
// to the centerline in the x-z plane.
func archWaveguideNormal(length, width, height, archHeight, z0 float64) shape {
	switch {
	case length <= 0:
		log.Log.ErrAndExit("ArchWaveguideNormal: length must be > 0, got %g", length)
	case width <= 0:
		log.Log.ErrAndExit("ArchWaveguideNormal: width must be > 0, got %g", width)
	case height <= 0:
		log.Log.ErrAndExit("ArchWaveguideNormal: height must be > 0, got %g", height)
	}

	halfL := length / 2
	centerZFn := func(x float64) float64 {
		t := (x + halfL) / length
		return z0 + archHeight*math.Sin(math.Pi*t)
	}
	dCenterZFn := func(x float64) float64 {
		t := (x + halfL) / length
		return archHeight * (math.Pi / length) * math.Cos(math.Pi*t)
	}
	ddCenterZFn := func(x float64) float64 {
		t := (x + halfL) / length
		return -archHeight * (math.Pi / length) * (math.Pi / length) * math.Sin(math.Pi*t)
	}

	base := normalThicknessWaveguide(length, width, height, centerZFn, dCenterZFn, ddCenterZFn)
	return newGuideShape(base.insideFn, base.voxelizer, newArchGuideGeometry(length, width, height, archHeight, z0))
}

// ellipsoid with given diameters
func ellipsoid(diamx, diamy, diamz float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		return sqr64(x/diamx)+sqr64(y/diamy)+sqr64(z/diamz) <= 0.25
	})
}

// superball with given diameter and shape parameter p
// A superball is defined by the inequality:
//
//	|x/r|^(2p) + |y/r|^(2p) + |z/r|^(2p) ≤ 1
//
// where r is the radius and p controls the shape:
//   - p > 1 gives a rounded cube
//   - p = 1 gives a sphere
//   - p = 0.5 gives an octahedron
//   - p <= 0 gives empty space
//
// for consistency with other shapes, diameter (2r) is used as parameter instead of radius
func superball(diameter, p float64) shape {
	if p <= 0 { // Yields empty shape
		return newSampledShape(func(x, y, z float64) bool { return false })
	}
	return newSampledShape(func(x, y, z float64) bool {
		norm := math.Pow(math.Abs(2*x/diameter), 2*p) +
			math.Pow(math.Abs(2*y/diameter), 2*p) +
			math.Pow(math.Abs(2*z/diameter), 2*p)
		return norm <= 1
	})
}

func ellipse(diamx, diamy float64) shape {
	return ellipsoid(diamx, diamy, math.Inf(1))
}

// 3D cone with base at z=0 and vertex at z=height.
func cone(diam, height float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		return (height-z)*z >= 0 && sqr64(x/diam)+sqr64(y/diam) <= 0.25*sqr64(1-z/height)
	})
}

func circle(diam float64) shape {
	return cylinder(diam, math.Inf(1))
}

// cylinder along z.
func cylinder(diam, height float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		return z <= height/2 && z >= -height/2 &&
			sqr64(x/diam)+sqr64(y/diam) <= 0.25
	})
}

// 3D Rectangular slab with given sides.
func cuboid(sidex, sidey, sidez float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		rx, ry, rz := sidex/2, sidey/2, sidez/2
		return x < rx && x > -rx && y < ry && y > -ry && z < rz && z > -rz
	})
}

// 2D Rectangle with given sides.
func rect(sidex, sidey float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		rx, ry := sidex/2, sidey/2
		return x < rx && x > -rx && y < ry && y > -ry
	})
}

// 2D triangle with given vertices using barycentric coordinates.
func triangle(x0, y0, x1, y1, x2, y2 float64) shape {
	denom := x0*(y1-y2) + x1*(y2-y0) + x2*(y0-y1) // 2 * area
	if denom == 0 {
		return newSampledShape(func(x, y, z float64) bool { return false })
	}
	A2m1 := 1 / denom

	Sc := A2m1 * (y0*x2 - x0*y2)
	Sx := A2m1 * (y2 - y0)
	Sy := A2m1 * (x0 - x2)

	Tc := A2m1 * (x0*y1 - y0*x1)
	Tx := A2m1 * (y0 - y1)
	Ty := A2m1 * (x1 - x0)

	return newSampledShape(func(x, y, z float64) bool {
		// barycentric coordinates
		s := Sc + Sx*x + Sy*y
		t := Tc + Tx*x + Ty*y
		return ((0 <= s) && (0 <= t) && (s+t <= 1))
	})
}

// eqTriangle creates an equilateral triangle with given side length, centered at origin.
func eqTriangle(side float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		c := math.Sqrt(3)
		return y > -side/(2*c) && y < x*c+side/c && y < -x*c+side/c
	})
}

// Rounded Equilateral triangle with given sides.
func rTriangle(side, diam float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		c := math.Sqrt(3)
		return y > -side/(2*c) && y < x*c+side/c && y < -x*c+side/c && math.Sqrt(sqr64(x)+sqr64(y)) < diam/2
	})
}

// hexagon with given sides.
func hexagon(side float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		a, b := math.Sqrt(3), math.Sqrt(3)*side
		return y < b/2 && y < -a*x+b && y > a*x-b && y > -b/2 && y > -a*x-b && y < a*x+b
	})
}

// diamond with given sides.
func diamond(sidex, sidey float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		a, b := sidey/sidex, sidey/2
		return y < a*x+b && y < -a*x+b && y > a*x-b && y > -a*x-b
	})
}

// squircle creates a 3D rounded rectangle (a generalized squircle) with specified side lengths and thickness.
func squircle(sidex, sidey, sidez, a float64) shape {
	// r := math.Min(sidex, sidey) / 2
	return newSampledShape(func(x, y, z float64) bool {
		normX := x / (sidex / 2)
		normY := y / (sidey / 2)

		value := normX*normX + normY*normY - a*normX*normX*normY*normY

		if math.Abs(x) > sidex/2 && math.Abs(y) > sidey/2 {
			return false
		}
		inSquircleXY := value <= 1
		rz := sidez / 2
		inThickness := z >= -rz && z <= rz
		return inSquircleXY && inThickness
	})
}

// 2D square with given side.
func square(side float64) shape {
	return rect(side, side)
}

// All cells with x-coordinate between a and b
func xRange(a, b float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		return x >= a && x < b
	})
}

// All cells with y-coordinate between a and b
func yRange(a, b float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		return y >= a && y < b
	})
}

// All cells with z-coordinate between a and b
func zRange(a, b float64) shape {
	return newSampledShape(func(x, y, z float64) bool {
		return z >= a && z < b
	})
}

// Cell layers #a (inclusive) up to #b (exclusive).
func layers(a, b int) shape {
	Nzi := GetMesh().Size()[Z]
	if a < 0 || a > Nzi || b < 0 || b < a {
		log.Log.ErrAndExit("layers %d:%d out of bounds (0 - %d)", a, b, Nzi)
	}
	c := GetMesh().CellSize()[Z]
	z1 := index2Coord(0, 0, a)[Z] - c/2
	z2 := index2Coord(0, 0, b)[Z] - c/2
	return zRange(z1, z2)
}

func layer(index int) shape {
	return layers(index, index+1)
}

// Single cell with given index
func cell(ix, iy, iz int) shape {
	c := GetMesh().CellSize()
	pos := index2Coord(ix, iy, iz)
	x1 := pos[X] - c[X]/2
	y1 := pos[Y] - c[Y]/2
	z1 := pos[Z] - c[Z]/2
	x2 := pos[X] + c[X]/2
	y2 := pos[Y] + c[Y]/2
	z2 := pos[Z] + c[Z]/2
	return newSampledShape(func(x, y, z float64) bool {
		return x > x1 && x < x2 &&
			y > y1 && y < y2 &&
			z > z1 && z < z2
	})
}

func universe() shape {
	return universeInner
}

var universeInner = newSampledShape(func(x, y, z float64) bool {
	return true
})

func imageShape(fname string) shape {
	r, err1 := fsutil.Open(fname)
	log.Log.PanicIfError(err1)
	defer func() {
		if err := r.Close(); err != nil {
			log.Log.PanicIfError(err)
		}
	}()
	img, _, err2 := image.Decode(r)
	log.Log.PanicIfError(err2)

	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y

	// decode image into bool matrix for fast pixel lookup
	inside := make([][]bool, height)
	for iy := range inside {
		inside[iy] = make([]bool, width)
	}
	for iy := 0; iy < height; iy++ {
		for ix := 0; ix < width; ix++ {
			r, g, b, a := img.At(ix, height-1-iy).RGBA()
			if a > 128 && r+g+b < (0xFFFF*3)/2 {
				inside[iy][ix] = true
			}
		}
	}

	// stretch the image onto the gridsize
	c := GetMesh().CellSize()
	cx, cy := c[X], c[Y]
	N := GetMesh().Size()
	nx, ny := float64(N[X]), float64(N[Y])
	w, h := float64(width), float64(height)
	return newSampledShape(func(x, y, z float64) bool {
		ix := int((w/nx)*(x/cx) + 0.5*w)
		iy := int((h/ny)*(y/cy) + 0.5*h)
		if ix < 0 || ix >= width || iy < 0 || iy >= height {
			return false
		}
		return inside[iy][ix]
	})
}

func grainRoughness(grainsize, zmin, zmax float64, seed int) shape {
	t := newTesselation(grainsize, 0, 256, int64(seed))
	return newSampledShape(func(x, y, z float64) bool {
		if z <= zmin {
			return true
		}
		if z >= zmax {
			return false
		}
		r := t.RegionOf(x, y, z)
		return (z-zmin)/(zmax-zmin) < (float64(r) / 256)
	})
}

// Transl returns a translated copy of the shape.
func (s shape) Transl(dx, dy, dz float64) shape {
	voxelizer := s.voxelizer
	if voxelizer != nil {
		voxelizer = translatedShapeVoxelizer{base: voxelizer, dx: dx, dy: dy, dz: dz}
	}
	var guide guideGeometry
	if s.guide != nil {
		guide = translatedGuideGeometry{base: s.guide, dx: dx, dy: dy, dz: dz}
	}
	return newGuideShape(func(x, y, z float64) bool {
		return s.contains(x-dx, y-dy, z-dz)
	}, voxelizer, guide)
}

// Repeat Infinitely repeats the shape with given period in x, y, z.
// A period of 0 or infinity means no repetition.

func (s shape) Repeat(periodX, periodY, periodZ float64) shape {
	return newDerivedShape(func(x, y, z float64) bool {
		return s.contains(fmod(x, periodX), fmod(y, periodY), fmod(z, periodZ))
	}, nil)
}

func fmod(a, b float64) float64 {
	if b == 0 || math.IsInf(b, 1) {
		return a
	}
	if math.Abs(a) > b/2 {
		return sign(a) * (math.Mod(math.Abs(a+b/2), b) - b/2)
	}
	return a
}

// Scale returns a scaled copy of the shape.
func (s shape) Scale(sx, sy, sz float64) shape {
	voxelizer := s.voxelizer
	if voxelizer != nil {
		voxelizer = scaledShapeVoxelizer{base: voxelizer, sx: sx, sy: sy, sz: sz}
	}
	return newDerivedShape(func(x, y, z float64) bool {
		return s.contains(x/sx, y/sy, z/sz)
	}, voxelizer)
}

// RotZ Rotates the shape around the Z-axis, over θ radians.
func (s shape) RotZ(θ float64) shape {
	cos := math.Cos(θ)
	sin := math.Sin(θ)
	return newDerivedShape(func(x, y, z float64) bool {
		xOut := x*cos + y*sin
		yOut := -x*sin + y*cos
		return s.contains(xOut, yOut, z)
	}, nil)
}

// RotY Rotates the shape around the Y-axis, over θ radians.
func (s shape) RotY(θ float64) shape {
	cos := math.Cos(θ)
	sin := math.Sin(θ)
	return newDerivedShape(func(x, y, z float64) bool {
		xOut := x*cos - z*sin
		zOut := x*sin + z*cos
		return s.contains(xOut, y, zOut)
	}, nil)
}

// RotX Rotates the shape around the X-axis, over θ radians.
func (s shape) RotX(θ float64) shape {
	cos := math.Cos(θ)
	sin := math.Sin(θ)
	return newDerivedShape(func(x, y, z float64) bool {
		yOut := y*cos + z*sin
		zOut := -y*sin + z*cos
		return s.contains(x, yOut, zOut)
	}, nil)
}

// Add Union of shapes a and b (logical OR).
func (s shape) Add(b shape) shape {
	return newDerivedShape(func(x, y, z float64) bool {
		return s.contains(x, y, z) || b.contains(x, y, z)
	}, nil)
}

// Intersect Intersection of shapes a and b (logical AND).
func (s shape) Intersect(b shape) shape {
	return newDerivedShape(func(x, y, z float64) bool {
		return s.contains(x, y, z) && b.contains(x, y, z)
	}, nil)
}

// Inverse (outside) of shape (logical NOT).
func (s shape) Inverse() shape {
	return newDerivedShape(func(x, y, z float64) bool {
		return !s.contains(x, y, z)
	}, nil)
}

// Sub Removes b from a (logical a AND NOT b)
func (s shape) Sub(b shape) shape {
	return newDerivedShape(func(x, y, z float64) bool {
		return s.contains(x, y, z) && !b.contains(x, y, z)
	}, nil)
}

// Xor Logical XOR of shapes a and b
func (s shape) Xor(b shape) shape {
	return newDerivedShape(func(x, y, z float64) bool {
		A, B := s.contains(x, y, z), b.contains(x, y, z)
		return (A || B) && (!A || !B)
	}, nil)
}

func sqr64(x float64) float64 { return x * x }
