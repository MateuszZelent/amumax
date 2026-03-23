package engine

import (
	"math"

	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
)

var (
	GuideProjectionEnabled = false
	GuideProjectionRefine  = 4
	GuideProjectionHalo    = 2
	GuideProjectionDS      = 0.0
	GuideProjectionDV      = 0.0
	GuideProjectionDW      = 0.0
	GuideProjectionUseCIC  = true
)

type guideProjectionROI struct {
	coarseMin  [3]int
	coarseSize [3]int
	fineSize   [3]int
	refine     int
	cell       [3]float64
	origin     [3]float64
	rho        []float64
}

type guideProjectionResult struct {
	hostGeom    *data.Slice
	hostFaces   *data.Slice
	linkX       []byte
	linkY       []byte
	linkZ       []byte
	empty       bool
	cutCells    int
	minPositive float32
	belowFloor  int
	roi         guideProjectionROI
}

func (g *geom) setGeomGuideProjected(s shape) (*data.Slice, bool) {
	result, ok := g.setGeomGuideProjectedHost(s)
	if !ok {
		return nil, false
	}

	data.Copy(g.Buffer, result.hostGeom)
	data.Copy(g.FaceBuffer, result.hostFaces)

	g.ensureLinkBuffer(&g.LinkX, len(result.linkX))
	g.ensureLinkBuffer(&g.LinkY, len(result.linkY))
	g.ensureLinkBuffer(&g.LinkZ, len(result.linkZ))
	g.LinkX.Upload(result.linkX)
	g.LinkY.Upload(result.linkY)
	g.LinkZ.Upload(result.linkZ)

	if result.cutCells > 0 {
		log.Log.Info(
			"Guide projection geometry: ROI coarse=%dx%dx%d fine=%dx%dx%d refine=%d, %d partial cells, minimum positive phi=%g",
			result.roi.coarseSize[X], result.roi.coarseSize[Y], result.roi.coarseSize[Z],
			result.roi.fineSize[X], result.roi.fineSize[Y], result.roi.fineSize[Z],
			result.roi.refine, result.cutCells, result.minPositive,
		)
	}
	if result.belowFloor > 0 {
		log.Log.Warn("Guide projection geometry: %d cells have phi < GeomPhiFloor=%g; exchange and DMI normalization will be clamped for stability", result.belowFloor, GeomPhiFloor)
	}
	if result.empty {
		log.Log.ErrAndExit("SetGeom: geometry completely empty")
	}

	return result.hostGeom, true
}

func (g *geom) setGeomGuideProjectedHost(s shape) (guideProjectionResult, bool) {
	var result guideProjectionResult
	if !guideProjectionRequested(s) {
		return result, false
	}

	refine := GuideProjectionRefine
	if refine <= 0 {
		log.Log.ErrAndExit("GuideProjectionRefine must be > 0, got %d", refine)
	}
	if GuideProjectionHalo < 0 {
		log.Log.ErrAndExit("GuideProjectionHalo must be >= 0, got %d", GuideProjectionHalo)
	}
	if GuideProjectionDS < 0 || GuideProjectionDV < 0 || GuideProjectionDW < 0 {
		log.Log.ErrAndExit("GuideProjectionDS/DV/DW must be >= 0")
	}

	roi, ok := newGuideProjectionROI(s.guide, refine, GuideProjectionHalo)
	if !ok {
		return result, false
	}

	log.Log.Info("Initializing guide projection geometry")
	projectGuideToROI(s.guide, &roi)
	clampGuideProjectionDensity(&roi)
	result = buildGuideProjectionResult(&roi)
	return result, true
}

func guideProjectionRequested(s shape) bool {
	return GuideProjectionEnabled && s.guide != nil && normalizedGeomMode() == "cutcell"
}

func newGuideProjectionROI(guide guideGeometry, refine, halo int) (guideProjectionROI, bool) {
	var roi guideProjectionROI
	xmin, xmax, ymin, ymax, zmin, zmax := guide.BoundingBox()
	size := GetMesh().Size()
	cell := GetMesh().CellSize()
	minEdge := meshMinEdge()

	x0, x1, ok := intersectingCellRange(size[X], minEdge[X], cell[X], xmin, xmax, halo)
	if !ok {
		return roi, false
	}
	y0, y1, ok := intersectingCellRange(size[Y], minEdge[Y], cell[Y], ymin, ymax, halo)
	if !ok {
		return roi, false
	}
	z0, z1, ok := intersectingCellRange(size[Z], minEdge[Z], cell[Z], zmin, zmax, halo)
	if !ok {
		return roi, false
	}

	roi.coarseMin = [3]int{x0, y0, z0}
	roi.coarseSize = [3]int{x1 - x0 + 1, y1 - y0 + 1, z1 - z0 + 1}
	roi.refine = refine
	roi.cell = [3]float64{cell[X] / float64(refine), cell[Y] / float64(refine), cell[Z] / float64(refine)}
	roi.fineSize = [3]int{
		roi.coarseSize[X] * refine,
		roi.coarseSize[Y] * refine,
		roi.coarseSize[Z] * refine,
	}
	roi.origin = [3]float64{
		minEdge[X] + float64(x0)*cell[X],
		minEdge[Y] + float64(y0)*cell[Y],
		minEdge[Z] + float64(z0)*cell[Z],
	}
	roi.rho = make([]float64, roi.fineSize[X]*roi.fineSize[Y]*roi.fineSize[Z])
	return roi, true
}

func projectGuideToROI(guide guideGeometry, roi *guideProjectionROI) {
	s0, s1 := guide.SRange()
	dsNominal := guideProjectionStep(GuideProjectionDS, roi)
	sCount, ds := quantizedSampleCountAndStep(s1-s0, dsNominal)

	for is := 0; is < sCount; is++ {
		s := s0 + (float64(is)+0.5)*ds
		frame := guide.FrameAtS(s)
		vMin, vMax, wMin, wMax := guide.CrossSectionBounds(s)
		vCount, dv := quantizedSampleCountAndStep(vMax-vMin, guideProjectionStep(GuideProjectionDV, roi))
		wCount, dw := quantizedSampleCountAndStep(wMax-wMin, guideProjectionStep(GuideProjectionDW, roi))
		dV := ds * dv * dw

		for iv := 0; iv < vCount; iv++ {
			v := vMin + (float64(iv)+0.5)*dv
			for iw := 0; iw < wCount; iw++ {
				w := wMin + (float64(iw)+0.5)*dw
				p := frame.R.add(frame.V.scale(v)).add(frame.W.scale(w))
				depositGuideProjection(roi, p, dV)
			}
		}
	}
}

func guideProjectionStep(config float64, roi *guideProjectionROI) float64 {
	if config > 0 {
		return config
	}
	return math.Min(roi.cell[X], math.Min(roi.cell[Y], roi.cell[Z]))
}

func quantizedSampleCountAndStep(span, nominal float64) (count int, step float64) {
	if !(span > 0) {
		return 1, 0
	}
	if nominal <= 0 {
		nominal = span
	}
	count = int(math.Ceil(span / nominal))
	if count < 1 {
		count = 1
	}
	step = span / float64(count)
	return count, step
}

func depositGuideProjection(roi *guideProjectionROI, p vec3, volume float64) {
	if GuideProjectionUseCIC {
		depositGuideProjectionCIC(roi, p, volume)
		return
	}
	depositGuideProjectionNearest(roi, p, volume)
}

func depositGuideProjectionNearest(roi *guideProjectionROI, p vec3, volume float64) {
	ix := int(math.Floor((p.X - roi.origin[X]) / roi.cell[X]))
	iy := int(math.Floor((p.Y - roi.origin[Y]) / roi.cell[Y]))
	iz := int(math.Floor((p.Z - roi.origin[Z]) / roi.cell[Z]))
	if !roi.containsFine(ix, iy, iz) {
		return
	}
	roi.rho[roi.fineIndex(ix, iy, iz)] += volume / roi.fineVolume()
}

func depositGuideProjectionCIC(roi *guideProjectionROI, p vec3, volume float64) {
	fx := (p.X-roi.origin[X])/roi.cell[X] - 0.5
	fy := (p.Y-roi.origin[Y])/roi.cell[Y] - 0.5
	fz := (p.Z-roi.origin[Z])/roi.cell[Z] - 0.5

	ix0 := int(math.Floor(fx))
	iy0 := int(math.Floor(fy))
	iz0 := int(math.Floor(fz))
	tx := fx - float64(ix0)
	ty := fy - float64(iy0)
	tz := fz - float64(iz0)
	wx := [2]float64{1 - tx, tx}
	wy := [2]float64{1 - ty, ty}
	wz := [2]float64{1 - tz, tz}
	scale := volume / roi.fineVolume()

	for oz := 0; oz < 2; oz++ {
		iz := iz0 + oz
		if !within(iz, roi.fineSize[Z]) {
			continue
		}
		for oy := 0; oy < 2; oy++ {
			iy := iy0 + oy
			if !within(iy, roi.fineSize[Y]) {
				continue
			}
			for ox := 0; ox < 2; ox++ {
				ix := ix0 + ox
				if !within(ix, roi.fineSize[X]) {
					continue
				}
				weight := wx[ox] * wy[oy] * wz[oz]
				if weight == 0 {
					continue
				}
				roi.rho[roi.fineIndex(ix, iy, iz)] += weight * scale
			}
		}
	}
}

func clampGuideProjectionDensity(roi *guideProjectionROI) {
	for i, rho := range roi.rho {
		switch {
		case rho <= 0:
			roi.rho[i] = 0
		case rho >= 1:
			roi.rho[i] = 1
		}
	}
}

func buildGuideProjectionResult(roi *guideProjectionROI) guideProjectionResult {
	hostGeom := data.NewSlice(1, Geometry.Mesh().Size())
	hostFaces := data.NewSlice(6, Geometry.Mesh().Size())
	geomValues := hostGeom.Host()[0]
	faceValues := hostFaces.Host()

	n := Geometry.Mesh().Size()
	total := n[X] * n[Y] * n[Z]
	linkX := make([]byte, total)
	linkY := make([]byte, total)
	linkZ := make([]byte, total)

	result := guideProjectionResult{
		hostGeom:    hostGeom,
		hostFaces:   hostFaces,
		linkX:       linkX,
		linkY:       linkY,
		linkZ:       linkZ,
		empty:       true,
		minPositive: 1,
		roi:         *roi,
	}

	r := roi.refine
	blockVol := float64(r * r * r)
	blockArea := float64(r * r)

	for cz := 0; cz < roi.coarseSize[Z]; cz++ {
		iz := roi.coarseMin[Z] + cz
		baseZ := cz * r
		for cy := 0; cy < roi.coarseSize[Y]; cy++ {
			iy := roi.coarseMin[Y] + cy
			baseY := cy * r
			for cx := 0; cx < roi.coarseSize[X]; cx++ {
				ix := roi.coarseMin[X] + cx
				baseX := cx * r
				idx := data.Index(n, ix, iy, iz)

				var sumPhi, fxm, fxp, fym, fyp, fzm, fzp float64
				for fz := 0; fz < r; fz++ {
					for fy := 0; fy < r; fy++ {
						fxm += roi.rhoAt(baseX, baseY+fy, baseZ+fz)
						fxp += roi.rhoAt(baseX+r-1, baseY+fy, baseZ+fz)
					}
				}
				for fz := 0; fz < r; fz++ {
					for fx := 0; fx < r; fx++ {
						fym += roi.rhoAt(baseX+fx, baseY, baseZ+fz)
						fyp += roi.rhoAt(baseX+fx, baseY+r-1, baseZ+fz)
					}
				}
				for fy := 0; fy < r; fy++ {
					for fx := 0; fx < r; fx++ {
						fzm += roi.rhoAt(baseX+fx, baseY+fy, baseZ)
						fzp += roi.rhoAt(baseX+fx, baseY+fy, baseZ+r-1)
					}
				}
				for fz := 0; fz < r; fz++ {
					for fy := 0; fy < r; fy++ {
						for fx := 0; fx < r; fx++ {
							sumPhi += roi.rhoAt(baseX+fx, baseY+fy, baseZ+fz)
						}
					}
				}

				phi := clampUnitFloat32(float32(sumPhi / blockVol))
				geomValues[idx] = phi
				faceValues[0][idx] = clampUnitFloat32(float32(fxm / blockArea))
				faceValues[1][idx] = clampUnitFloat32(float32(fxp / blockArea))
				faceValues[2][idx] = clampUnitFloat32(float32(fym / blockArea))
				faceValues[3][idx] = clampUnitFloat32(float32(fyp / blockArea))
				faceValues[4][idx] = clampUnitFloat32(float32(fzm / blockArea))
				faceValues[5][idx] = clampUnitFloat32(float32(fzp / blockArea))

				if phi > 0 {
					result.empty = false
					if phi < result.minPositive {
						result.minPositive = phi
					}
					if phi < float32(GeomPhiFloor) {
						result.belowFloor++
					}
				}
				if phi > 0 && phi < 1 {
					result.cutCells++
				}

				if ix+1 < n[X] {
					linkX[idx] = quantizeLinkFraction(clampUnitFloat32(float32(roi.interfaceFractionX(cx, cy, cz))))
				}
				if iy+1 < n[Y] {
					linkY[idx] = quantizeLinkFraction(clampUnitFloat32(float32(roi.interfaceFractionY(cx, cy, cz))))
				}
				if iz+1 < n[Z] {
					linkZ[idx] = quantizeLinkFraction(clampUnitFloat32(float32(roi.interfaceFractionZ(cx, cy, cz))))
				}
			}
		}
	}

	if hasPeriodicBoundaries() {
		result.linkX, result.linkY, result.linkZ = buildLinksFromFaces(hostFaces)
	}

	return result
}

func hasPeriodicBoundaries() bool {
	pbc := GetMesh().PBC()
	return pbc[X] != 0 || pbc[Y] != 0 || pbc[Z] != 0
}

func buildLinksFromFaces(faceHost *data.Slice) ([]byte, []byte, []byte) {
	n := GetMesh().Size()
	total := n[X] * n[Y] * n[Z]
	linkX := make([]byte, total)
	linkY := make([]byte, total)
	linkZ := make([]byte, total)
	face := faceHost.Host()
	pbc := GetMesh().PBC()

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

	return linkX, linkY, linkZ
}

func intersectingCellRange(n int, gridMin, d, boxMin, boxMax float64, halo int) (start, end int, ok bool) {
	gridMax := gridMin + float64(n)*d
	if boxMax <= gridMin || boxMin >= gridMax {
		return 0, 0, false
	}
	start = int(math.Floor((boxMin - gridMin) / d))
	end = int(math.Ceil((boxMax-gridMin)/d)) - 1
	if start < 0 {
		start = 0
	}
	if end >= n {
		end = n - 1
	}
	start -= halo
	end += halo
	if start < 0 {
		start = 0
	}
	if end >= n {
		end = n - 1
	}
	return start, end, start <= end
}

func meshMinEdge() [3]float64 {
	cell := GetMesh().CellSize()
	r := index2Coord(0, 0, 0)
	return [3]float64{
		r[X] - cell[X]/2,
		r[Y] - cell[Y]/2,
		r[Z] - cell[Z]/2,
	}
}

func (roi *guideProjectionROI) fineVolume() float64 {
	return roi.cell[X] * roi.cell[Y] * roi.cell[Z]
}

func (roi *guideProjectionROI) fineIndex(ix, iy, iz int) int {
	return (iz*roi.fineSize[Y]+iy)*roi.fineSize[X] + ix
}

func (roi *guideProjectionROI) containsFine(ix, iy, iz int) bool {
	return within(ix, roi.fineSize[X]) && within(iy, roi.fineSize[Y]) && within(iz, roi.fineSize[Z])
}

func (roi *guideProjectionROI) rhoAt(ix, iy, iz int) float64 {
	if !roi.containsFine(ix, iy, iz) {
		return 0
	}
	return roi.rho[roi.fineIndex(ix, iy, iz)]
}

func (roi *guideProjectionROI) interfaceFractionX(cx, cy, cz int) float64 {
	if cx+1 >= roi.coarseSize[X] {
		return 0
	}
	baseX := cx * roi.refine
	baseY := cy * roi.refine
	baseZ := cz * roi.refine
	var sum float64
	for fz := 0; fz < roi.refine; fz++ {
		for fy := 0; fy < roi.refine; fy++ {
			left := roi.rhoAt(baseX+roi.refine-1, baseY+fy, baseZ+fz)
			right := roi.rhoAt(baseX+roi.refine, baseY+fy, baseZ+fz)
			sum += math.Min(left, right)
		}
	}
	return sum / float64(roi.refine*roi.refine)
}

func (roi *guideProjectionROI) interfaceFractionY(cx, cy, cz int) float64 {
	if cy+1 >= roi.coarseSize[Y] {
		return 0
	}
	baseX := cx * roi.refine
	baseY := cy * roi.refine
	baseZ := cz * roi.refine
	var sum float64
	for fz := 0; fz < roi.refine; fz++ {
		for fx := 0; fx < roi.refine; fx++ {
			left := roi.rhoAt(baseX+fx, baseY+roi.refine-1, baseZ+fz)
			right := roi.rhoAt(baseX+fx, baseY+roi.refine, baseZ+fz)
			sum += math.Min(left, right)
		}
	}
	return sum / float64(roi.refine*roi.refine)
}

func (roi *guideProjectionROI) interfaceFractionZ(cx, cy, cz int) float64 {
	if cz+1 >= roi.coarseSize[Z] {
		return 0
	}
	baseX := cx * roi.refine
	baseY := cy * roi.refine
	baseZ := cz * roi.refine
	var sum float64
	for fy := 0; fy < roi.refine; fy++ {
		for fx := 0; fx < roi.refine; fx++ {
			left := roi.rhoAt(baseX+fx, baseY+fy, baseZ+roi.refine-1)
			right := roi.rhoAt(baseX+fx, baseY+fy, baseZ+roi.refine)
			sum += math.Min(left, right)
		}
	}
	return sum / float64(roi.refine*roi.refine)
}

func within(i, n int) bool {
	return i >= 0 && i < n
}
