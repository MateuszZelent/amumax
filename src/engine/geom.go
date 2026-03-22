package engine

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/mesh"
)

func init() {
	Geometry.init()
	GeomPhi = newScalarField("geom_phi", "", "Cell fill fraction (0..1); explicit alias of geom for cut-cell diagnostics", setGeomPhi)
	GeomThicknessZ = newScalarField("geom_thickness_z", "m", "Numerical material thickness obtained by summing cell fill fractions along z", setGeomThicknessZ)
	GeomFx = newScalarField("geom_fx", "", "Positive x-face fill fraction between a cell and its +x neighbor (0..1)", func(dst *data.Slice) { setGeomPositiveFace(dst, 1) })
	GeomFy = newScalarField("geom_fy", "", "Positive y-face fill fraction between a cell and its +y neighbor (0..1)", func(dst *data.Slice) { setGeomPositiveFace(dst, 3) })
	GeomFz = newScalarField("geom_fz", "", "Positive z-face fill fraction between a cell and its +z neighbor (0..1)", func(dst *data.Slice) { setGeomPositiveFace(dst, 5) })
	GeomLinkX = newScalarField("geom_link_x", "", "Shared x-interface fraction used by cut-cell exchange between a cell and its +x neighbor (0..1)", func(dst *data.Slice) { setGeomLink(dst, Geometry.LinkX) })
	GeomLinkY = newScalarField("geom_link_y", "", "Shared y-interface fraction used by cut-cell exchange between a cell and its +y neighbor (0..1)", func(dst *data.Slice) { setGeomLink(dst, Geometry.LinkY) })
	GeomLinkZ = newScalarField("geom_link_z", "", "Shared z-interface fraction used by cut-cell exchange between a cell and its +z neighbor (0..1)", func(dst *data.Slice) { setGeomLink(dst, Geometry.LinkZ) })
	GeomFaceX = newScalarField("geom_face_x", "", "Average x-face fill fraction (0..1)", setGeomFaceX)
	GeomFaceY = newScalarField("geom_face_y", "", "Average y-face fill fraction (0..1)", setGeomFaceY)
	GeomFaceZ = newScalarField("geom_face_z", "", "Average z-face fill fraction (0..1)", setGeomFaceZ)
}

var (
	Geometry       geom
	edgeSmooth     int = 0 // disabled by default
	GeomMode           = "cutcell"
	GeomTol            = 1e-3
	GeomMaxDepth       = 4
	GeomPhiFloor       = 0.05
	GeomPhi        ScalarField
	GeomThicknessZ ScalarField
	GeomFx         ScalarField
	GeomFy         ScalarField
	GeomFz         ScalarField
	GeomLinkX      ScalarField
	GeomLinkY      ScalarField
	GeomLinkZ      ScalarField
	GeomFaceX      ScalarField
	GeomFaceY      ScalarField
	GeomFaceZ      ScalarField
)

func normalizedGeomMode() string {
	mode := strings.ToLower(strings.TrimSpace(GeomMode))
	if mode == "" {
		return "cutcell"
	}
	return mode
}

func geomCutCellEnabled(s shape) bool {
	switch normalizedGeomMode() {
	case "cutcell":
		return s.voxelizer != nil
	case "legacy":
		return false
	default:
		log.Log.ErrAndExit(`GeomMode: unsupported mode %q (expected "legacy" or "cutcell")`, GeomMode)
		return false
	}
}

func geomUsesCutCell(s shape) bool {
	switch normalizedGeomMode() {
	case "cutcell":
		if s.voxelizer == nil {
			log.Log.Warn("GeomMode=cutcell requested, but shape has no voxelizer; falling back to legacy geometry")
			return false
		}
		return true
	case "legacy":
		return false
	default:
		log.Log.ErrAndExit(`GeomMode: unsupported mode %q (expected "legacy" or "cutcell")`, GeomMode)
		return false
	}
}

func (g *geom) usesCutCell() bool {
	return geomCutCellEnabled(g.shape)
}

type geom struct {
	info
	Buffer     *data.Slice
	FaceBuffer *data.Slice
	LinkX      *cuda.Bytes
	LinkY      *cuda.Bytes
	LinkZ      *cuda.Bytes
	shape      shape
}

func (g *geom) init() {
	g.Buffer = nil
	g.FaceBuffer = nil
	g.LinkX = nil
	g.LinkY = nil
	g.LinkZ = nil
	g.info = info{1, "geom", ""}
	declROnly("geom", g, "Cell fill fraction (0..1)")
}

func (g *geom) HasLinks() bool {
	return g.LinkX != nil && g.LinkY != nil && g.LinkZ != nil
}

func (g *geom) ClearLinks() {
	if g.LinkX != nil {
		g.LinkX.Free()
		g.LinkX = nil
	}
	if g.LinkY != nil {
		g.LinkY.Free()
		g.LinkY = nil
	}
	if g.LinkZ != nil {
		g.LinkZ.Free()
		g.LinkZ = nil
	}
}

func (g *geom) Gpu() *data.Slice {
	if g.Buffer == nil {
		g.Buffer = data.NilSlice(1, g.Mesh().Size())
	}
	return g.Buffer
}

func (g *geom) Slice() (*data.Slice, bool) {
	s := g.Gpu()
	if s.IsNil() {
		buffer := cuda.Buffer(g.NComp(), g.Mesh().Size())
		cuda.Memset(buffer, 1)
		return buffer, true
	}
	return s, false
}

func (g *geom) FaceSlice() (*data.Slice, bool) {
	if g.FaceBuffer == nil || g.FaceBuffer.IsNil() || g.FaceBuffer.Size() != g.Mesh().Size() {
		buffer := cuda.Buffer(6, g.Mesh().Size())
		cuda.Memset(buffer, 1, 1, 1, 1, 1, 1)
		return buffer, true
	}
	return g.FaceBuffer, false
}

func (g *geom) EvalTo(dst *data.Slice) { evalTo(g, dst) }

var _ Quantity = &Geometry

func (g *geom) average() []float64 {
	s, r := g.Slice()
	if r {
		defer cuda.Recycle(s)
	}
	return sAverageUniverse(s)
}

func (g *geom) Average() float64 { return g.average()[0] }

func setGeomPhi(dst *data.Slice) {
	Geometry.EvalTo(dst)
}

func setGeomPositiveFace(dst *data.Slice, comp int) {
	faces, recycle := Geometry.FaceSlice()
	if recycle {
		defer cuda.Recycle(faces)
	}

	hostFaces := faces.HostCopy()
	faceValues := hostFaces.Host()[comp]
	hostDst := data.NewSlice(1, dst.Size())
	copy(hostDst.Host()[0], faceValues)
	data.Copy(dst, hostDst)
}

func setGeomLink(dst *data.Slice, link *cuda.Bytes) {
	hostDst := data.NewSlice(1, dst.Size())
	if link == nil || link.Len == 0 {
		data.Copy(dst, hostDst)
		return
	}

	values := make([]byte, link.Len)
	link.Download(values)
	dstValues := hostDst.Host()[0]
	for i, value := range values {
		dstValues[i] = float32(value) / 255
	}
	data.Copy(dst, hostDst)
}

func setGeomThicknessZ(dst *data.Slice) {
	geom, recycle := Geometry.Slice()
	if recycle {
		defer cuda.Recycle(geom)
	}

	hostGeom := geom.HostCopy()
	geomValues := hostGeom.Scalars()
	hostDst := data.NewSlice(1, dst.Size())
	thicknessValues := hostDst.Scalars()
	n := Geometry.Mesh().Size()
	dz := float32(Geometry.Mesh().CellSize()[Z])

	for iy := 0; iy < n[Y]; iy++ {
		for ix := 0; ix < n[X]; ix++ {
			var thickness float32
			for iz := 0; iz < n[Z]; iz++ {
				thickness += geomValues[iz][iy][ix] * dz
			}
			for iz := 0; iz < n[Z]; iz++ {
				thicknessValues[iz][iy][ix] = thickness
			}
		}
	}

	data.Copy(dst, hostDst)
}

func setGeomFaceX(dst *data.Slice) { setGeomFaceAxis(dst, 0, 1) }
func setGeomFaceY(dst *data.Slice) { setGeomFaceAxis(dst, 2, 3) }
func setGeomFaceZ(dst *data.Slice) { setGeomFaceAxis(dst, 4, 5) }

func setGeomFaceAxis(dst *data.Slice, negativeComp, positiveComp int) {
	faces, recycle := Geometry.FaceSlice()
	if recycle {
		defer cuda.Recycle(faces)
	}

	hostFaces := faces.HostCopy()
	faceValues := hostFaces.Host()
	hostDst := data.NewSlice(1, dst.Size())
	avgValues := hostDst.Host()[0]
	negative := faceValues[negativeComp]
	positive := faceValues[positiveComp]

	for i := range avgValues {
		avgValues[i] = 0.5 * (negative[i] + positive[i])
	}

	data.Copy(dst, hostDst)
}

func (g *geom) setGeom(s shape) {
	setBusy(true)
	defer setBusy(false)

	if s.isNil() {
		// TODO: would be nice not to save volume if entirely filled
		s = universeInner
	}

	g.shape = s
	if g.Gpu().IsNil() {
		g.Buffer = cuda.NewSlice(1, g.Mesh().Size())
	}
	if g.FaceBuffer == nil || g.FaceBuffer.IsNil() || g.FaceBuffer.Size() != g.Mesh().Size() {
		g.FaceBuffer = cuda.NewSlice(6, g.Mesh().Size())
	}
	useCutCell := geomUsesCutCell(s)
	if !useCutCell {
		g.ClearLinks()
	}

	var host *data.Slice
	empty := true

	if useCutCell {
		var hostFaces *data.Slice
		var cutCells int
		var minPositive float32
		var belowFloor int
		host, hostFaces, empty, cutCells, minPositive, belowFloor = g.setGeomCutCellHost(s)
		data.Copy(g.Buffer, host)
		data.Copy(g.FaceBuffer, hostFaces)
		g.rebuildLinks(hostFaces)
		if cutCells > 0 {
			log.Log.Info("Cut-cell geometry: %d partial cells, minimum positive phi=%g", cutCells, minPositive)
		}
		if belowFloor > 0 {
			log.Log.Warn("Cut-cell geometry: %d cells have phi < GeomPhiFloor=%g; exchange and DMI normalization will be clamped for stability", belowFloor, GeomPhiFloor)
		}
	} else {
		host, empty = g.setGeomLegacyHost(s)
		data.Copy(g.Buffer, host)
		g.rebuildFaceBuffer()
	}

	if empty {
		log.Log.ErrAndExit("SetGeom: geometry completely empty")
	}

	// M inside geom but previously outside needs to be re-inited
	needupload := false
	geomlist := host.Host()[0]
	mhost := NormMag.Buffer().HostCopy()
	m := mhost.Host()
	rng := rand.New(rand.NewSource(0))
	for i := range m[0] {
		if geomlist[i] != 0 {
			mx, my, mz := m[X][i], m[Y][i], m[Z][i]
			if mx == 0 && my == 0 && mz == 0 {
				needupload = true
				rnd := randomDir(rng)
				m[X][i], m[Y][i], m[Z][i] = float32(rnd[X]), float32(rnd[Y]), float32(rnd[Z])
			}
		}
	}
	if needupload {
		data.Copy(NormMag.Buffer(), mhost)
	}

	NormMag.normalize() // removes m outside vol
}

func (g *geom) setGeomCutCellHost(s shape) (*data.Slice, *data.Slice, bool, int, float32, int) {
	hostGeom := data.NewSlice(1, g.Gpu().Size())
	hostFaces := data.NewSlice(6, g.Mesh().Size())
	geomValues := hostGeom.Host()[0]
	faceValues := hostFaces.Host()
	n := g.Mesh().Size()
	empty := true
	minPositive := float32(1)
	cutCells := 0
	belowFloor := 0

	log.Log.Info("Initializing geometry")
	for iz := 0; iz < n[Z]; iz++ {
		pct := 100 * (iz + 1) / n[Z]
		fmt.Fprintf(os.Stderr, "\rInitializing geometry: %3d%%", pct)
		for iy := 0; iy < n[Y]; iy++ {
			for ix := 0; ix < n[X]; ix++ {
				idx := data.Index(n, ix, iy, iz)
				metrics := s.voxelizer.cellMetrics(boundsFromIndex(ix, iy, iz))
				phi := metrics.VolumeFraction
				geomValues[idx] = phi
				for comp, value := range metrics.FaceFraction {
					faceValues[comp][idx] = value
				}
				if phi > 0 {
					empty = false
					if phi < minPositive {
						minPositive = phi
					}
					if phi < float32(GeomPhiFloor) {
						belowFloor++
					}
				}
				if phi > 0 && phi < 1 {
					cutCells++
				}
			}
		}
	}
	fmt.Fprintf(os.Stderr, "\n")

	return hostGeom, hostFaces, empty, cutCells, minPositive, belowFloor
}

func (g *geom) setGeomLegacyHost(s shape) (*data.Slice, bool) {
	host := data.NewSlice(1, g.Gpu().Size())
	v := host.Scalars()
	n := g.Mesh().Size()
	c := g.Mesh().CellSize()
	cx, cy, cz := c[X], c[Y], c[Z]
	empty := true

	log.Log.Info("Initializing geometry")
	for iz := 0; iz < n[Z]; iz++ {
		pct := 100 * (iz + 1) / n[Z]
		fmt.Fprintf(os.Stderr, "\rInitializing geometry: %3d%%", pct)
		for iy := 0; iy < n[Y]; iy++ {
			for ix := 0; ix < n[X]; ix++ {
				r := index2Coord(ix, iy, iz)
				x0, y0, z0 := r[X], r[Y], r[Z]

				// check if center and all vertices lie inside or all outside
				allIn, allOut := true, true
				if s.contains(x0, y0, z0) {
					allOut = false
				} else {
					allIn = false
				}

				if edgeSmooth != 0 { // center is sufficient if we're not really smoothing
					for _, Δx := range []float64{-cx / 2, cx / 2} {
						for _, Δy := range []float64{-cy / 2, cy / 2} {
							for _, Δz := range []float64{-cz / 2, cz / 2} {
								if s.contains(x0+Δx, y0+Δy, z0+Δz) { // inside
									allOut = false
								} else {
									allIn = false
								}
							}
						}
					}
				}

				switch {
				case allIn:
					v[iz][iy][ix] = 1
					empty = false
				case allOut:
					v[iz][iy][ix] = 0
				default:
					v[iz][iy][ix] = g.cellVolume(ix, iy, iz)
					empty = empty && (v[iz][iy][ix] == 0)
				}
			}
		}
	}
	fmt.Fprintf(os.Stderr, "\n")

	return host, empty
}

func edgeSmoothSamples() int {
	if edgeSmooth <= 0 {
		return 1
	}
	return edgeSmooth
}

// Sample edgeSmooth^3 points inside the cell to estimate its volume.
func (g *geom) cellVolume(ix, iy, iz int) float32 {
	if g.usesCutCell() {
		return g.shape.voxelizer.cellMetrics(boundsFromIndex(ix, iy, iz)).VolumeFraction
	}

	r := index2Coord(ix, iy, iz)
	x0, y0, z0 := r[X], r[Y], r[Z]

	c := Geometry.Mesh().CellSize()
	cx, cy, cz := c[X], c[Y], c[Z]
	s := Geometry.shape
	var vol float32

	N := edgeSmooth
	S := float64(edgeSmooth)

	for dx := 0; dx < N; dx++ {
		Δx := -cx/2 + (cx / (2 * S)) + (cx/S)*float64(dx)
		for dy := 0; dy < N; dy++ {
			Δy := -cy/2 + (cy / (2 * S)) + (cy/S)*float64(dy)
			for dz := 0; dz < N; dz++ {
				Δz := -cz/2 + (cz / (2 * S)) + (cz/S)*float64(dz)

				if s.contains(x0+Δx, y0+Δy, z0+Δz) { // inside
					vol++
				}
			}
		}
	}
	return vol / float32(N*N*N)
}

func samplePositiveFaceFill(axis, ix, iy, iz int) float32 {
	if Geometry.usesCutCell() {
		return Geometry.shape.voxelizer.cellMetrics(boundsFromIndex(ix, iy, iz)).Face(axis, true)
	}

	r := index2Coord(ix, iy, iz)
	x0, y0, z0 := r[X], r[Y], r[Z]

	c := Geometry.Mesh().CellSize()
	cx, cy, cz := c[X], c[Y], c[Z]
	s := Geometry.shape
	N := edgeSmoothSamples()
	S := float64(N)
	eps := []float64{cx, cy, cz}[axis] * 1e-12
	if eps == 0 {
		eps = 1e-12
	}

	var fill float32
	switch axis {
	case X:
		x := x0 + cx/2 - eps
		for dy := 0; dy < N; dy++ {
			y := y0 - cy/2 + (cy / (2 * S)) + (cy/S)*float64(dy)
			for dz := 0; dz < N; dz++ {
				z := z0 - cz/2 + (cz / (2 * S)) + (cz/S)*float64(dz)
				if s.contains(x, y, z) {
					fill++
				}
			}
		}
	case Y:
		y := y0 + cy/2 - eps
		for dx := 0; dx < N; dx++ {
			x := x0 - cx/2 + (cx / (2 * S)) + (cx/S)*float64(dx)
			for dz := 0; dz < N; dz++ {
				z := z0 - cz/2 + (cz / (2 * S)) + (cz/S)*float64(dz)
				if s.contains(x, y, z) {
					fill++
				}
			}
		}
	case Z:
		z := z0 + cz/2 - eps
		for dx := 0; dx < N; dx++ {
			x := x0 - cx/2 + (cx / (2 * S)) + (cx/S)*float64(dx)
			for dy := 0; dy < N; dy++ {
				y := y0 - cy/2 + (cy / (2 * S)) + (cy/S)*float64(dy)
				if s.contains(x, y, z) {
					fill++
				}
			}
		}
	default:
		log.Log.ErrAndExit("samplePositiveFaceFill: invalid axis %d", axis)
	}

	return fill / float32(N*N)
}

func sampleNegativeBoundaryFaceFill(axis, ix, iy, iz int) float32 {
	if Geometry.usesCutCell() {
		return Geometry.shape.voxelizer.cellMetrics(boundsFromIndex(ix, iy, iz)).Face(axis, false)
	}

	r := index2Coord(ix, iy, iz)
	x0, y0, z0 := r[X], r[Y], r[Z]

	c := Geometry.Mesh().CellSize()
	cx, cy, cz := c[X], c[Y], c[Z]
	s := Geometry.shape
	N := edgeSmoothSamples()
	S := float64(N)
	eps := []float64{cx, cy, cz}[axis] * 1e-12
	if eps == 0 {
		eps = 1e-12
	}

	var fill float32
	switch axis {
	case X:
		x := x0 - cx/2 + eps
		for dy := 0; dy < N; dy++ {
			y := y0 - cy/2 + (cy / (2 * S)) + (cy/S)*float64(dy)
			for dz := 0; dz < N; dz++ {
				z := z0 - cz/2 + (cz / (2 * S)) + (cz/S)*float64(dz)
				if s.contains(x, y, z) {
					fill++
				}
			}
		}
	case Y:
		y := y0 - cy/2 + eps
		for dx := 0; dx < N; dx++ {
			x := x0 - cx/2 + (cx / (2 * S)) + (cx/S)*float64(dx)
			for dz := 0; dz < N; dz++ {
				z := z0 - cz/2 + (cz / (2 * S)) + (cz/S)*float64(dz)
				if s.contains(x, y, z) {
					fill++
				}
			}
		}
	case Z:
		z := z0 - cz/2 + eps
		for dx := 0; dx < N; dx++ {
			x := x0 - cx/2 + (cx / (2 * S)) + (cx/S)*float64(dx)
			for dy := 0; dy < N; dy++ {
				y := y0 - cy/2 + (cy / (2 * S)) + (cy/S)*float64(dy)
				if s.contains(x, y, z) {
					fill++
				}
			}
		}
	default:
		log.Log.ErrAndExit("sampleNegativeBoundaryFaceFill: invalid axis %d", axis)
	}

	return fill / float32(N*N)
}

func (g *geom) rebuildFaceBuffer() {
	if g.FaceBuffer == nil || g.FaceBuffer.IsNil() || g.FaceBuffer.Size() != g.Mesh().Size() {
		g.FaceBuffer = cuda.NewSlice(6, g.Mesh().Size())
	}

	host := data.NewSlice(6, g.Mesh().Size())
	face := host.Host()
	n := g.Mesh().Size()
	pbc := g.Mesh().PBC()

	if g.usesCutCell() {
		for iz := 0; iz < n[Z]; iz++ {
			for iy := 0; iy < n[Y]; iy++ {
				for ix := 0; ix < n[X]; ix++ {
					idx := data.Index(n, ix, iy, iz)
					metrics := g.shape.voxelizer.cellMetrics(boundsFromIndex(ix, iy, iz))
					for comp, value := range metrics.FaceFraction {
						face[comp][idx] = value
					}
				}
			}
		}
		data.Copy(g.FaceBuffer, host)
		g.rebuildLinks(host)
		return
	}

	for iz := 0; iz < n[Z]; iz++ {
		for iy := 0; iy < n[Y]; iy++ {
			for ix := 0; ix < n[X]; ix++ {
				idx := data.Index(n, ix, iy, iz)
				face[1][idx] = samplePositiveFaceFill(X, ix, iy, iz)
				face[3][idx] = samplePositiveFaceFill(Y, ix, iy, iz)
				face[5][idx] = samplePositiveFaceFill(Z, ix, iy, iz)
			}
		}
	}

	for iz := 0; iz < n[Z]; iz++ {
		for iy := 0; iy < n[Y]; iy++ {
			for ix := 0; ix < n[X]; ix++ {
				idx := data.Index(n, ix, iy, iz)

				if ix == 0 {
					if pbc[X] != 0 {
						face[0][idx] = face[1][data.Index(n, n[X]-1, iy, iz)]
					} else {
						face[0][idx] = sampleNegativeBoundaryFaceFill(X, ix, iy, iz)
					}
				} else {
					face[0][idx] = face[1][data.Index(n, ix-1, iy, iz)]
				}

				if iy == 0 {
					if pbc[Y] != 0 {
						face[2][idx] = face[3][data.Index(n, ix, n[Y]-1, iz)]
					} else {
						face[2][idx] = sampleNegativeBoundaryFaceFill(Y, ix, iy, iz)
					}
				} else {
					face[2][idx] = face[3][data.Index(n, ix, iy-1, iz)]
				}

				if iz == 0 {
					if pbc[Z] != 0 {
						face[4][idx] = face[5][data.Index(n, ix, iy, n[Z]-1)]
					} else {
						face[4][idx] = sampleNegativeBoundaryFaceFill(Z, ix, iy, iz)
					}
				} else {
					face[4][idx] = face[5][data.Index(n, ix, iy, iz-1)]
				}
			}
		}
	}

	data.Copy(g.FaceBuffer, host)
	g.rebuildLinks(host)
}

func quantizeLinkFraction(v float32) byte {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	default:
		return byte(v*255 + 0.5)
	}
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func (g *geom) ensureLinkBuffer(dst **cuda.Bytes, size int) {
	if *dst != nil && (*dst).Len == size {
		return
	}
	if *dst != nil {
		(*dst).Free()
	}
	*dst = cuda.NewBytes(size)
}

func (g *geom) rebuildLinks(faceHost *data.Slice) {
	if !g.usesCutCell() {
		g.ClearLinks()
		return
	}

	n := g.Mesh().Size()
	total := n[X] * n[Y] * n[Z]
	g.ensureLinkBuffer(&g.LinkX, total)
	g.ensureLinkBuffer(&g.LinkY, total)
	g.ensureLinkBuffer(&g.LinkZ, total)

	face := faceHost.Host()
	linkX := make([]byte, total)
	linkY := make([]byte, total)
	linkZ := make([]byte, total)
	pbc := g.Mesh().PBC()

	for iz := 0; iz < n[Z]; iz++ {
		for iy := 0; iy < n[Y]; iy++ {
			for ix := 0; ix < n[X]; ix++ {
				idx := data.Index(n, ix, iy, iz)

				if ix+1 < n[X] {
					right := data.Index(n, ix+1, iy, iz)
					linkX[idx] = quantizeLinkFraction(minFloat32(face[1][idx], face[0][right]))
				} else if pbc[X] != 0 {
					wrap := data.Index(n, 0, iy, iz)
					linkX[idx] = quantizeLinkFraction(minFloat32(face[1][idx], face[0][wrap]))
				}

				if iy+1 < n[Y] {
					front := data.Index(n, ix, iy+1, iz)
					linkY[idx] = quantizeLinkFraction(minFloat32(face[3][idx], face[2][front]))
				} else if pbc[Y] != 0 {
					wrap := data.Index(n, ix, 0, iz)
					linkY[idx] = quantizeLinkFraction(minFloat32(face[3][idx], face[2][wrap]))
				}

				if iz+1 < n[Z] {
					up := data.Index(n, ix, iy, iz+1)
					linkZ[idx] = quantizeLinkFraction(minFloat32(face[5][idx], face[4][up]))
				} else if pbc[Z] != 0 {
					wrap := data.Index(n, ix, iy, 0)
					linkZ[idx] = quantizeLinkFraction(minFloat32(face[5][idx], face[4][wrap]))
				}
			}
		}
	}

	g.LinkX.Upload(linkX)
	g.LinkY.Upload(linkY)
	g.LinkZ.Upload(linkZ)
}

func (g *geom) GetCell(ix, iy, iz int) float64 {
	return float64(cuda.GetCell(g.Gpu(), 0, ix, iy, iz))
}

func (g *geom) shift(dx int) {
	// empty mask, nothing to do
	if g == nil || g.Buffer == nil || g.Buffer.IsNil() {
		return
	}
	if g.HasLinks() {
		g.setGeom(g.shape)
		return
	}

	// allocated mask: shift
	s := g.Buffer
	s2 := cuda.Buffer(1, g.Mesh().Size())
	defer cuda.Recycle(s2)
	newv := float32(1) // initially fill edges with 1's
	cuda.ShiftX(s2, s, dx, newv, newv)
	data.Copy(s, s2)

	n := GetMesh().Size()
	x1, x2 := shiftDirtyRange(dx)

	for iz := 0; iz < n[Z]; iz++ {
		for iy := 0; iy < n[Y]; iy++ {
			for ix := x1; ix < x2; ix++ {
				r := index2Coord(ix, iy, iz) // includes shift
				if !g.shape.contains(r[X], r[Y], r[Z]) {
					cuda.SetCell(g.Buffer, 0, ix, iy, iz, 0) // a bit slowish, but hardly reached
				}
			}
		}
	}

	if !g.shape.isNil() && g.FaceBuffer != nil && !g.FaceBuffer.IsNil() {
		g.rebuildFaceBuffer()
	}
}

func (g *geom) shiftY(dy int) {
	// empty mask, nothing to do
	if g == nil || g.Buffer == nil || g.Buffer.IsNil() {
		return
	}
	if g.HasLinks() {
		g.setGeom(g.shape)
		return
	}

	// allocated mask: shift
	s := g.Buffer
	s2 := cuda.Buffer(1, g.Mesh().Size())
	defer cuda.Recycle(s2)
	newv := float32(1) // initially fill edges with 1's
	cuda.ShiftY(s2, s, dy, newv, newv)
	data.Copy(s, s2)

	n := GetMesh().Size()
	y1, y2 := shiftDirtyRange(dy)

	for iz := 0; iz < n[Z]; iz++ {
		for ix := 0; ix < n[X]; ix++ {
			for iy := y1; iy < y2; iy++ {
				r := index2Coord(ix, iy, iz) // includes shift
				if !g.shape.contains(r[X], r[Y], r[Z]) {
					cuda.SetCell(g.Buffer, 0, ix, iy, iz, 0) // a bit slowish, but hardly reached
				}
			}
		}
	}

	if !g.shape.isNil() && g.FaceBuffer != nil && !g.FaceBuffer.IsNil() {
		g.rebuildFaceBuffer()
	}
}

// x range that needs to be refreshed after shift over dx
func shiftDirtyRange(dx int) (x1, x2 int) {
	nx := GetMesh().Size()[X]
	log.AssertMsg(dx != 0, "Invalid shift: dx must not be zero in shiftDirtyRange")

	if dx < 0 {
		x1 = nx + dx
		x2 = nx
	} else {
		x1 = 0
		x2 = dx
	}
	return
}

func (g *geom) Mesh() *mesh.Mesh { return GetMesh() }
