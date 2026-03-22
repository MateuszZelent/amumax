package engine

import (
	"sync"
	"sync/atomic"

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
	demagBoundaryRevision uint64
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
	mesh     [3]int
	halo     int
	radius   int
	refine   int
	revision uint64
}

const demagBoundaryEps = 1e-4

func invalidateDemagBoundaryPlan() {
	demagBoundaryCache.mu.Lock()
	defer demagBoundaryCache.mu.Unlock()
	atomic.AddUint64(&demagBoundaryRevision, 1)
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
		mesh:     Geometry.Mesh().Size(),
		halo:     halo,
		radius:   radius,
		refine:   refine,
		revision: atomic.LoadUint64(&demagBoundaryRevision),
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
	cell := Geometry.Mesh().CellSize()
	base := make([]bool, len(maskValues))
	active := make([]bool, len(maskValues))
	plan := demagBoundaryPlan{
		mask:    hostMask,
		stencil: buildDemagStencil(cfg.radius),
	}

	if !Geometry.usesCutCell() || Geometry.shape.voxelizer == nil {
		plan.maskGPU = cuda.GPUCopy(hostMask)
		return plan
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

	if pbc[X] != 0 || pbc[Y] != 0 || pbc[Z] != 0 {
		log.Log.Warn("DemagBoundaryCorr currently supports only non-periodic meshes; skipping local demag correction for PBC != 0")
		plan.maskGPU = cuda.GPUCopy(hostMask)
		return plan
	}

	coarseGrid := [3]int{1, 1, 1}
	fineGrid := [3]int{cfg.refine, cfg.refine, cfg.refine}
	for axis := 0; axis < 3; axis++ {
		if n[axis] > 1 {
			coarseGrid[axis] = cfg.radius + 1
			fineGrid[axis] = (cfg.radius + 1) * cfg.refine
		}
	}
	coarseKernel := newDemagKernelLookup(coarseGrid, cell)
	fineCell := [3]float64{
		cell[X] / float64(cfg.refine),
		cell[Y] / float64(cfg.refine),
		cell[Z] / float64(cfg.refine),
	}
	fineKernel := newDemagKernelLookup(fineGrid, fineCell)

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

	refinedCells := make(map[int]refinedCellWeights, len(support))
	for idx := range support {
		ix, iy, iz := demagLinearToCoord(idx, n)
		refinedCells[idx] = buildRefinedCellWeights(boundsFromIndex(ix, iy, iz), float64(geomValues[idx]), cfg.refine)
	}

	targetHost := make([]int32, len(plan.targetIdx))
	sourceHost := make([]int32, len(plan.targetIdx)*len(plan.stencil))
	for i := range sourceHost {
		sourceHost[i] = -1
	}
	tensorHost := data.NewSlice(1, [3]int{len(plan.targetIdx) * len(plan.stencil) * 6, 1, 1})
	tensorValues := tensorHost.Host()[0]

	for t, targetIdx := range plan.targetIdx {
		targetHost[t] = int32(targetIdx)
		targetWeights := refinedCells[targetIdx]
		if targetWeights.total <= demagBoundaryEps {
			continue
		}
		tix, tiy, tiz := demagLinearToCoord(targetIdx, n)

		for s, delta := range plan.stencil {
			sourceIdx, ok := demagWrappedIndex(n, pbc, tix+delta[X], tiy+delta[Y], tiz+delta[Z])
			if !ok || !active[sourceIdx] {
				continue
			}
			sourceWeights := refinedCells[sourceIdx]
			if sourceWeights.total <= demagBoundaryEps {
				continue
			}
			sourceHost[t*len(plan.stencil)+s] = int32(sourceIdx)
			tensor := demagRefinedCorrectionTensor(targetWeights, sourceWeights, delta, cfg.refine, fineKernel, coarseKernel)
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
	if radius < 0 {
		return nil
	}
	stencil := make([][3]int, 0, (2*radius+1)*(2*radius+1)*(2*radius+1))
	for dz := -radius; dz <= radius; dz++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
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
		log.Log.Warn("DemagBoundaryCorr uses an experimental sparse GPU-side local demag boundary correction on the boundary shell. It precomputes a refined local correction tensor (including self terms) per shell target and applies it after the FFT demag field.")
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

type refinedSubcellWeight struct {
	rx, ry, rz int
	weight     float64
}

type refinedCellWeights struct {
	total float64
	subs  []refinedSubcellWeight
}

type demagKernelLookup struct {
	size [3]int
	xx   [][][]float32
	xy   [][][]float32
	xz   [][][]float32
	yy   [][][]float32
	yz   [][][]float32
	zz   [][][]float32
}

func newDemagKernelLookup(grid [3]int, cellSize [3]float64) demagKernelLookup {
	kernel := mag.DemagKernel(grid, [3]int{}, cellSize, DemagAccuracy, CacheDir, true)
	return demagKernelLookup{
		size: kernel[X][X].Size(),
		xx:   kernel[X][X].Scalars(),
		xy:   kernel[X][Y].Scalars(),
		xz:   demagKernelScalars(kernel[X][Z]),
		yy:   kernel[Y][Y].Scalars(),
		yz:   demagKernelScalars(kernel[Y][Z]),
		zz:   kernel[Z][Z].Scalars(),
	}
}

func demagKernelScalars(s *data.Slice) [][][]float32 {
	if s == nil {
		return nil
	}
	return s.Scalars()
}

func demagKernelWrapOffset(offset, size int) int {
	for offset < 0 {
		offset += size
	}
	for offset >= size {
		offset -= size
	}
	return offset
}

func (k demagKernelLookup) tensor(dx, dy, dz int) [6]float64 {
	ix := demagKernelWrapOffset(dx, k.size[X])
	iy := demagKernelWrapOffset(dy, k.size[Y])
	iz := demagKernelWrapOffset(dz, k.size[Z])
	var out [6]float64
	out[0] = float64(k.xx[iz][iy][ix])
	out[1] = float64(k.xy[iz][iy][ix])
	if k.xz != nil {
		out[2] = float64(k.xz[iz][iy][ix])
	}
	out[3] = float64(k.yy[iz][iy][ix])
	if k.yz != nil {
		out[4] = float64(k.yz[iz][iy][ix])
	}
	out[5] = float64(k.zz[iz][iy][ix])
	return out
}

func splitBoundsAxis(min, max float64, idx, refine int) (float64, float64) {
	span := (max - min) / float64(refine)
	start := min + float64(idx)*span
	return start, start + span
}

func subcellBounds(bounds cellBounds, rx, ry, rz, refine int) cellBounds {
	xMin, xMax := splitBoundsAxis(bounds.xMin, bounds.xMax, rx, refine)
	yMin, yMax := splitBoundsAxis(bounds.yMin, bounds.yMax, ry, refine)
	zMin, zMax := splitBoundsAxis(bounds.zMin, bounds.zMax, rz, refine)
	return cellBounds{xMin: xMin, xMax: xMax, yMin: yMin, yMax: yMax, zMin: zMin, zMax: zMax}
}

func buildRefinedCellWeights(bounds cellBounds, coarsePhi float64, refine int) refinedCellWeights {
	if coarsePhi <= demagBoundaryEps || refine < 1 {
		return refinedCellWeights{}
	}

	count := refine * refine * refine
	weights := refinedCellWeights{
		subs: make([]refinedSubcellWeight, 0, count),
	}

	if coarsePhi >= 1-demagBoundaryEps {
		for rz := 0; rz < refine; rz++ {
			for ry := 0; ry < refine; ry++ {
				for rx := 0; rx < refine; rx++ {
					weights.subs = append(weights.subs, refinedSubcellWeight{rx: rx, ry: ry, rz: rz, weight: 1})
				}
			}
		}
		weights.total = float64(len(weights.subs))
		return weights
	}

	for rz := 0; rz < refine; rz++ {
		for ry := 0; ry < refine; ry++ {
			for rx := 0; rx < refine; rx++ {
				subBounds := subcellBounds(bounds, rx, ry, rz, refine)
				phi := float64(Geometry.shape.voxelizer.cellMetrics(subBounds).VolumeFraction)
				if phi <= demagBoundaryEps {
					continue
				}
				if phi > 1-demagBoundaryEps {
					phi = 1
				}
				weights.subs = append(weights.subs, refinedSubcellWeight{rx: rx, ry: ry, rz: rz, weight: phi})
				weights.total += phi
			}
		}
	}

	return weights
}

func demagAddScaledTensor(dst *[6]float64, src [6]float64, scale float64) {
	for i := 0; i < 6; i++ {
		dst[i] += scale * src[i]
	}
}

func demagRefinedCorrectionTensor(target, source refinedCellWeights, delta [3]int, refine int, fineKernel, coarseKernel demagKernelLookup) [6]float64 {
	var fine [6]float64
	for _, tSub := range target.subs {
		wt := tSub.weight / target.total
		for _, sSub := range source.subs {
			ws := sSub.weight / source.total
			dx := delta[X]*refine + (tSub.rx - sSub.rx)
			dy := delta[Y]*refine + (tSub.ry - sSub.ry)
			dz := delta[Z]*refine + (tSub.rz - sSub.rz)
			demagAddScaledTensor(&fine, fineKernel.tensor(dx, dy, dz), wt*ws)
		}
	}

	coarse := coarseKernel.tensor(delta[X], delta[Y], delta[Z])
	for i := 0; i < 6; i++ {
		fine[i] -= coarse[i]
	}
	return fine
}
