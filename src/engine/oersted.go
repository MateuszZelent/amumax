package engine

// Oersted field calculation from current density.
//
// The Oersted field B_oersted is computed from a spatial current density
// distribution J_oersted and a time-dependent scalar multiplier I_time:
//
//   B_oersted(r, t) = I_time(t) * B_base(r)
//
// where B_base is the Oersted field for J_oersted with I_time=1, computed
// via FFT convolution with a real-space Biot-Savart kernel:
//
//   K(r) = mu0/(4pi) * dV * r / |r|^3
//   B = J × K   (cross-product convolution)
//
// The expensive FFT is only recomputed when J_oersted or the mesh changes.
// At each time step, only a cheap Madd2 scaling is performed.

import (
	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/mag"
)

var (
	// JOersted is the spatial current density distribution for the Oersted field.
	JOersted = newExcitation("J_oersted", "A/m2", "Current density for Oersted field calculation")

	// IOersted is the time-dependent scalar multiplier for the Oersted field.
	// Dimensionless (default 1). Example: IOersted = sin(2*pi*1e9*t)
	IOersted = newScalarExcitation("I_oersted", "", "Time-dependent multiplier for Oersted field")

	// BOersted is the Oersted field output quantity.
	BOersted = newVectorField("B_oersted", "T", "Oersted field from current density", setOerstedField)

	// EnableOersted enables/disables the Oersted field contribution to B_eff.
	EnableOersted = false

	oerstedWarnedNoJ bool

	// Cached base field (Oersted field for I_time=1)
	oerstedBase *data.Slice

	// Cached convolution engine
	oerstedConv *cuda.OerstedConvolution

	// Cache invalidation: last known revision of J_oersted
	lastJOerstedRev uint64

	// Cache invalidation: last known mesh signature
	lastOerstedMeshSize [3]int
	lastOerstedMeshPBC  [3]int
	lastOerstedCellSize [3]float64
)

func init() {
	declVar("EnableOersted", &EnableOersted, "Enables/disables Oersted field (default=false)")
	IOersted.Set(1) // default amplitude = 1 (user only needs to set J_oersted)
}

// setOerstedField computes the Oersted field and stores it in dst.
// Used for Save(B_oersted) and GUI preview. Does NOT require EnableOersted=true
// so the field is always visible when J_oersted is set.
func setOerstedField(dst *data.Slice) {
	cuda.Zero(dst)
	if JOersted.isZero() {
		return
	}
	amp := float32(IOersted.average())
	if amp == 0 {
		return
	}
	ensureOerstedBase()
	cuda.Madd2(dst, dst, oerstedBase, 1, amp)
}

// addOerstedField adds the Oersted field contribution to dst.
// Called from setEffectiveField.
func addOerstedField(dst *data.Slice) {
	if !EnableOersted {
		return
	}

	// Check if J_oersted evaluates to zero (not just structurally zero).
	// isZero() can give false positives for time-dependent expressions
	// like vector(0,0, Jdc + 0*sinc(t)) where the LUT is overwritten
	// by the dynamic updater, even though the actual value is non-zero.
	avg := JOersted.average()
	allZero := true
	for _, v := range avg {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero && JOersted.isZero() {
		if !oerstedWarnedNoJ {
			log.Log.Warn("EnableOersted=true but J_oersted is zero. Set J_oersted explicitly (e.g. J_oersted = J).")
			oerstedWarnedNoJ = true
		}
		return
	}
	oerstedWarnedNoJ = false

	// Get time-dependent amplitude
	amp := float32(IOersted.average())
	if amp == 0 {
		return
	}

	// Ensure base field is up to date
	ensureOerstedBase()

	// dst += amp * oerstedBase
	cuda.Madd2(dst, dst, oerstedBase, 1, amp)
}

// ensureOerstedBase recomputes the cached Oersted base field if needed.
func ensureOerstedBase() {
	meshSize := GetMesh().Size()
	meshPBC := GetMesh().PBC()
	meshCell := GetMesh().CellSize()

	needRecompute := oerstedBase == nil ||
		oerstedConv == nil ||
		meshSize != lastOerstedMeshSize ||
		meshPBC != lastOerstedMeshPBC ||
		meshCell != lastOerstedCellSize ||
		JOersted.Revision() != lastJOerstedRev

	if !needRecompute {
		return
	}

	recomputeOerstedBase()
}

// recomputeOerstedBase performs the full FFT convolution to compute the
// Oersted base field from J_oersted.
func recomputeOerstedBase() {
	meshSize := GetMesh().Size()
	meshPBC := GetMesh().PBC()
	meshCell := GetMesh().CellSize()

	// (Re)create convolution engine if mesh changed
	if oerstedConv == nil || meshSize != lastOerstedMeshSize || meshPBC != lastOerstedMeshPBC || meshCell != lastOerstedCellSize {
		if oerstedConv != nil {
			oerstedConv.Free()
		}
		// Compute real-space Biot-Savart kernel on host
		kernel := mag.OerstedKernel(meshSize, meshCell, meshPBC)
		// Create convolution engine (FFTs kernel, stores on GPU)
		oerstedConv = cuda.NewOersted(meshSize, kernel)
		// Free host-side kernel
		for i := range kernel {
			kernel[i].Free()
		}
	}

	// (Re)allocate base field buffer if needed
	if oerstedBase == nil || oerstedBase.Size() != meshSize {
		if oerstedBase != nil {
			oerstedBase.Free()
		}
		oerstedBase = cuda.NewSlice(3, meshSize)
	}

	// Get current density
	jSlice, rec := JOersted.Slice()
	if rec {
		defer cuda.Recycle(jSlice)
	}

	// Get geometry mask
	vol := Geometry.Gpu()

	// Compute B_base = Oersted(J_oersted) with I_time=1
	oerstedConv.Exec(oerstedBase, jSlice, vol)

	// Update cache signatures
	lastJOerstedRev = JOersted.Revision()
	lastOerstedMeshSize = meshSize
	lastOerstedMeshPBC = meshPBC
	lastOerstedCellSize = meshCell
}

// FreeOersted releases cached Oersted resources.
// Called when mesh is resized.
func FreeOersted() {
	if oerstedConv != nil {
		oerstedConv.Free()
		oerstedConv = nil
	}
	if oerstedBase != nil {
		oerstedBase.Free()
		oerstedBase = nil
	}
	lastJOerstedRev = 0
	lastOerstedMeshSize = [3]int{}
	lastOerstedMeshPBC = [3]int{}
	lastOerstedCellSize = [3]float64{}
	oerstedWarnedNoJ = false
}
