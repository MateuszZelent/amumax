package cuda

import (
	"math"

	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/mesh"
)

func SetHopfIndexDensity_TwoPointStencil(h, m *data.Slice, mesh mesh.MeshLike) {
	N := m.Size()

	// Create buffers to store emergent field F and vector potential A
	F := Buffer(3, N)
	defer Recycle(F)
	A := Buffer(3, N)
	defer Recycle(A)

	// Get Hopf index density F · A
	SetEmergentMagneticField_TwoPointStencil(F, m, mesh)
	SetVectorPotential(A, F, mesh)
	AddDotProduct(h, 1.0, F, A)
}

// Sets the emergent magnetic field F_i = (1/8π) ε_{ijk} m · (∂m/∂x_j × ∂m/∂x_k)
// See hopf-emergentmagneticfield.cu
func SetEmergentMagneticField_TwoPointStencil(F, m *data.Slice, mesh mesh.MeshLike) {
	cellsize := mesh.CellSize()
	N := F.Size()
	log.AssertMsg(m.Size() == N, "Hopf index: size mismatch")
	cfg := make3DConf(N)
	icycz := float32(1.0 / (cellsize[Y] * cellsize[Z]))
	iczcx := float32(1.0 / (cellsize[Z] * cellsize[X]))
	icxcy := float32(1.0 / (cellsize[X] * cellsize[Y]))
	prefactor := float32(1.0 / (8 * math.Pi))

	kSetemergentmagneticfieldtwopointAsync(F.DevPtr(X), F.DevPtr(Y), F.DevPtr(Z),
		m.DevPtr(X), m.DevPtr(Y), m.DevPtr(Z), prefactor,
		icycz, iczcx, icxcy, N[X], N[Y], N[Z], mesh.PBCCode(), cfg)
}
