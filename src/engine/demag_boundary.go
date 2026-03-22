package engine

import (
	"math"
	"sync"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/mag"
)

var (
	DemagBoundaryCorr   = false
	DemagBoundaryRadius = 1
	DemagBoundaryRefine = 4
	DemagBoundaryHalo   = 1
	DemagBoundaryTol    = 1e-3

	DemagBoundaryShell       ScalarField
	DemagBoundaryTargetCount *ScalarValue

	demagBoundaryWarnOnce sync.Once
	demagBoundaryCache    demagBoundaryPlanCache
)

func init() {
	DemagBoundaryShell = newScalarField("demag_boundary_shell", "", "Boundary-shell mask for planned local demag correction (1 inside shell, 0 outside)", setDemagBoundaryShell)
	DemagBoundaryTargetCount = newScalarValue("demag_boundary_target_count", "", "Number of cells currently included in the demag boundary shell", getDemagBoundaryTargetCount)
}

func setDemagBoundaryShell(dst *data.Slice) {
	plan := getDemagBoundaryPlan()
	data.Copy(dst, plan.mask)
}

type demagBoundaryPlan struct {
	mask         *data.Slice
	maskGPU      *data.Slice
	targetIdx    []int
	targetIdxGPU *cuda.Int32s
	sourceIdxGPU *cuda.Int32s
	tensorGPU    *data.Slice
	stencil      [][3]int
}

type demagBoundaryPlanCache struct {
	mu    sync.RWMutex
	valid bool
	cfg   demagBoundaryConfig
	plan  demagBoundaryPlan
}

type demagBoundaryConfig struct {
	mesh   [3]int
	halo   int
	radius int
	refine int
}

const demagBoundaryEps = 1e-4

func invalidateDemagBoundaryPlan() {
	demagBoundaryCache.mu.Lock()
	defer demagBoundaryCache.mu.Unlock()
	freeDemagBoundaryPlanGPU(&demagBoundaryCache.plan)
	demagBoundaryCache.valid = false
	demagBoundaryCache.plan = demagBoundaryPlan{}
	demagBoundaryWarnOnce = sync.Once{}
}

func freeDemagBoundaryPlanGPU(plan *demagBoundaryPlan) {
	if plan.maskGPU != nil {
		plan.maskGPU.Free()
		plan.maskGPU = nil
	}
	if plan.targetIdxGPU != nil {
		plan.targetIdxGPU.Free()
		plan.targetIdxGPU = nil
	}
	if plan.sourceIdxGPU != nil {
		plan.sourceIdxGPU.Free()
		plan.sourceIdxGPU = nil
	}
	if plan.tensorGPU != nil {
		plan.tensorGPU.Free()
		plan.tensorGPU = nil
	}
}

func getDemagBoundaryTargetCount() float64 {
	return float64(len(getDemagBoundaryPlan().targetIdx))
}

func getDemagBoundaryPlan() demagBoundaryPlan {
	demagBoundaryCache.mu.RLock()
	cfg := currentDemagBoundaryConfig()
	if demagBoundaryCache.valid && demagBoundaryCache.cfg == cfg {
		plan := demagBoundaryCache.plan
		demagBoundaryCache.mu.RUnlock()
		return plan
	}
	demagBoundaryCache.mu.RUnlock()

	demagBoundaryCache.mu.Lock()
	defer demagBoundaryCache.mu.Unlock()
	cfg = currentDemagBoundaryConfig()
	if demagBoundaryCache.valid && demagBoundaryCache.cfg == cfg {
		return demagBoundaryCache.plan
	}
	freeDemagBoundaryPlanGPU(&demagBoundaryCache.plan)
	demagBoundaryCache.plan = buildDemagBoundaryPlanHost()
	demagBoundaryCache.cfg = cfg
	demagBoundaryCache.valid = true
	return demagBoundaryCache.plan
}

func currentDemagBoundaryConfig() demagBoundaryConfig {
	halo := DemagBoundaryHalo
	if halo < 0 {
		halo = 0
	}
	radius := DemagBoundaryRadius
	if radius < 0 {
		radius = 0
	}
	refine := DemagBoundaryRefine
	if refine < 1 {
		refine = 1
	}
	return demagBoundaryConfig{
		mesh:   Geometry.Mesh().Size(),
		halo:   halo,
		radius: radius,
		refine: refine,
	}
}

func buildDemagBoundaryPlanHost() demagBoundaryPlan {
	cfg := currentDemagBoundaryConfig()
	hostMask := data.NewSlice(1, Geometry.Mesh().Size())

	geom, recycleGeom := Geometry.Slice()
	if recycleGeom {
		defer cuda.Recycle(geom)
	}
	faces, recycleFaces := Geometry.FaceSlice()
	if recycleFaces {
		defer cuda.Recycle(faces)
	}

	hostGeom := geom.HostCopy()
	hostFaces := faces.HostCopy()
	geomValues := hostGeom.Host()[0]
	faceValues := hostFaces.Host()
	maskValues := hostMask.Host()[0]
	n := Geometry.Mesh().Size()
	pbc := Geometry.Mesh().PBC()
	base := make([]bool, len(maskValues))
	active := make([]bool, len(maskValues))
	offsets := make([]data.Vector, len(maskValues))
	plan := demagBoundaryPlan{
		mask:    hostMask,
		stencil: buildDemagStencil(cfg.radius),
	}

	for idx, phi := range geomValues {
		if phi <= demagBoundaryEps {
			continue
		}
		active[idx] = true
		if phi < 1-demagBoundaryEps {
			base[idx] = true
			continue
		}
		for comp := 0; comp < 6; comp++ {
			face := faceValues[comp][idx]
			if face > demagBoundaryEps && face < 1-demagBoundaryEps {
				base[idx] = true
				break
			}
		}
	}

	if cfg.halo == 0 {
		for idx, marked := range base {
			if marked {
				maskValues[idx] = 1
				plan.targetIdx = append(plan.targetIdx, idx)
			}
		}
	} else {
		for iz := 0; iz < n[Z]; iz++ {
			for iy := 0; iy < n[Y]; iy++ {
				for ix := 0; ix < n[X]; ix++ {
					idx := data.Index(n, ix, iy, iz)
					if !active[idx] {
						continue
					}
					if demagShellHasBoundaryNeighbor(base, n, ix, iy, iz, cfg.halo) {
						maskValues[idx] = 1
						plan.targetIdx = append(plan.targetIdx, idx)
					}
				}
			}
		}
	}

	if len(plan.targetIdx) == 0 || len(plan.stencil) == 0 {
		plan.maskGPU = cuda.GPUCopy(hostMask)
		return plan
	}

	support := make(map[int]struct{}, len(plan.targetIdx))
	for _, idx := range plan.targetIdx {
		support[idx] = struct{}{}
		ix, iy, iz := demagLinearToCoord(idx, n)
		for _, delta := range plan.stencil {
			neighborIdx, ok := demagWrappedIndex(n, pbc, ix+delta[X], iy+delta[Y], iz+delta[Z])
			if !ok || !active[neighborIdx] {
				continue
			}
			support[neighborIdx] = struct{}{}
		}
	}

	for idx := range support {
		ix, iy, iz := demagLinearToCoord(idx, n)
		offsets[idx] = demagCellCentroidOffset(ix, iy, iz, geomValues[idx], cfg.refine)
	}

	targetHost := make([]int32, len(plan.targetIdx))
	sourceHost := make([]int32, len(plan.targetIdx)*len(plan.stencil))
	for i := range sourceHost {
		sourceHost[i] = -1
	}
	tensorHost := data.NewSlice(1, [3]int{len(plan.targetIdx) * len(plan.stencil) * 6, 1, 1})
	tensorValues := tensorHost.Host()[0]
	cell := Geometry.Mesh().CellSize()

	for t, targetIdx := range plan.targetIdx {
		targetHost[t] = int32(targetIdx)
		tix, tiy, tiz := demagLinearToCoord(targetIdx, n)
		targetOffset := offsets[targetIdx]

		for s, delta := range plan.stencil {
			sourceIdx, ok := demagWrappedIndex(n, pbc, tix+delta[X], tiy+delta[Y], tiz+delta[Z])
			if !ok || !active[sourceIdx] {
				continue
			}
			sourceHost[t*len(plan.stencil)+s] = int32(sourceIdx)
			sourceOffset := offsets[sourceIdx]
			coarseR := vector(
				-float64(delta[X])*cell[X],
				-float64(delta[Y])*cell[Y],
				-float64(delta[Z])*cell[Z],
			)
			fineR := coarseR.Add(targetOffset).Sub(sourceOffset)
			tensor := demagTensorDifference(fineR, coarseR, cell[X]*cell[Y]*cell[Z])
			base := (t*len(plan.stencil) + s) * 6
			for comp := 0; comp < 6; comp++ {
				tensorValues[base+comp] = float32(tensor[comp])
			}
		}
	}

	plan.maskGPU = cuda.GPUCopy(hostMask)
	plan.targetIdxGPU = cuda.NewInt32s(len(targetHost))
	plan.targetIdxGPU.Upload(targetHost)
	plan.sourceIdxGPU = cuda.NewInt32s(len(sourceHost))
	plan.sourceIdxGPU.Upload(sourceHost)
	plan.tensorGPU = cuda.GPUCopy(tensorHost)
	return plan
}

func buildDemagStencil(radius int) [][3]int {
	if radius <= 0 {
		return nil
	}
	stencil := make([][3]int, 0, (2*radius+1)*(2*radius+1)*(2*radius+1)-1)
	for dz := -radius; dz <= radius; dz++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx == 0 && dy == 0 && dz == 0 {
					continue
				}
				stencil = append(stencil, [3]int{dx, dy, dz})
			}
		}
	}
	return stencil
}

func demagShellHasBoundaryNeighbor(base []bool, n [3]int, ix, iy, iz, halo int) bool {
	xMin := intMax(ix-halo, 0)
	xMax := intMin(ix+halo, n[X]-1)
	yMin := intMax(iy-halo, 0)
	yMax := intMin(iy+halo, n[Y]-1)
	zMin := intMax(iz-halo, 0)
	zMax := intMin(iz+halo, n[Z]-1)

	for zz := zMin; zz <= zMax; zz++ {
		for yy := yMin; yy <= yMax; yy++ {
			for xx := xMin; xx <= xMax; xx++ {
				if base[data.Index(n, xx, yy, zz)] {
					return true
				}
			}
		}
	}
	return false
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func applyDemagBoundaryCorrection(dst, vol *data.Slice, ms cuda.MSlice) {
	if !DemagBoundaryCorr {
		return
	}
	plan := getDemagBoundaryPlan()
	demagBoundaryWarnOnce.Do(func() {
		log.Log.Warn("DemagBoundaryCorr uses an experimental sparse GPU-side local demag boundary correction on the boundary shell. It precomputes a local correction tensor per shell target and applies it after the FFT demag field.")
		log.Log.Info("Demag boundary shell currently covers %d cells", len(plan.targetIdx))
	})
	if len(plan.targetIdx) == 0 || len(plan.stencil) == 0 {
		return
	}
	cuda.AddDemagBoundaryCorr(dst, NormMag.Buffer(), ms, vol, plan.targetIdxGPU, plan.sourceIdxGPU, plan.tensorGPU, len(plan.stencil))
}

func demagLinearToCoord(idx int, n [3]int) (ix, iy, iz int) {
	plane := n[X] * n[Y]
	iz = idx / plane
	rest := idx - iz*plane
	iy = rest / n[X]
	ix = rest - iy*n[X]
	return
}

func demagWrappedIndex(n, pbc [3]int, ix, iy, iz int) (int, bool) {
	coords := [3]int{ix, iy, iz}
	for axis := 0; axis < 3; axis++ {
		switch {
		case coords[axis] < 0:
			if pbc[axis] == 0 {
				return 0, false
			}
			coords[axis] += n[axis]
		case coords[axis] >= n[axis]:
			if pbc[axis] == 0 {
				return 0, false
			}
			coords[axis] -= n[axis]
		}
	}
	return data.Index(n, coords[X], coords[Y], coords[Z]), true
}

func demagCellCentroidOffset(ix, iy, iz int, phi float32, refine int) data.Vector {
	if phi <= demagBoundaryEps || phi >= 1-demagBoundaryEps {
		return data.Vector{}
	}
	if refine < 1 {
		refine = 1
	}
	bounds := boundsFromIndex(ix, iy, iz)
	center := index2Coord(ix, iy, iz)
	var sum data.Vector
	var count int

	for rz := 0; rz < refine; rz++ {
		tz := (float64(rz) + 0.5) / float64(refine)
		for ry := 0; ry < refine; ry++ {
			ty := (float64(ry) + 0.5) / float64(refine)
			for rx := 0; rx < refine; rx++ {
				tx := (float64(rx) + 0.5) / float64(refine)
				x, y, z := bounds.samplePoint(tx, ty, tz)
				if Geometry.shape.contains(x, y, z) {
					sum = sum.Add(vector(x-center[X], y-center[Y], z-center[Z]))
					count++
				}
			}
		}
	}
	if count == 0 {
		return data.Vector{}
	}
	return sum.Div(float64(count))
}

func demagTensorDifference(fineR, coarseR data.Vector, cellVolume float64) [6]float64 {
	fine := demagTensorFromR(fineR, cellVolume)
	coarse := demagTensorFromR(coarseR, cellVolume)
	for i := 0; i < 6; i++ {
		fine[i] -= coarse[i]
	}
	return fine
}

func demagTensorFromR(r data.Vector, cellVolume float64) [6]float64 {
	r2 := r.Dot(r)
	if r2 < 1e-30 {
		return [6]float64{}
	}
	rLen := math.Sqrt(r2)
	r3 := r2 * rLen
	r5 := r3 * r2
	prefactor := (mag.Mu0 / (4 * math.Pi)) * cellVolume

	xx := prefactor * ((3*r[X]*r[X])/r5 - 1/r3)
	xy := prefactor * ((3 * r[X] * r[Y]) / r5)
	xz := prefactor * ((3 * r[X] * r[Z]) / r5)
	yy := prefactor * ((3*r[Y]*r[Y])/r5 - 1/r3)
	yz := prefactor * ((3 * r[Y] * r[Z]) / r5)
	zz := prefactor * ((3*r[Z]*r[Z])/r5 - 1/r3)
	return [6]float64{xx, xy, xz, yy, yz, zz}
}
