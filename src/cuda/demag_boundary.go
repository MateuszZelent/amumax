package cuda

import (
	"github.com/MathieuMoalic/amumax/src/data"
)

// AddDemagBoundaryCorr adds a sparse local demag boundary correction on top of the
// FFT demag field. The correction is applied only to targetIdx entries and uses
// sourceIdx + precomputed symmetric 3x3 tensors stored as 6 components per source.
func AddDemagBoundaryCorr(B, m *data.Slice, Msat MSlice, phi *data.Slice, targetIdx, sourceIdx *Int32s, tensor *data.Slice, stencilCount int) {
	if targetIdx == nil || targetIdx.Len == 0 || sourceIdx == nil || sourceIdx.Len == 0 || tensor == nil || stencilCount <= 0 {
		return
	}
	cfg := make1DConf(targetIdx.Len)
	kAddDemagBoundaryCorrAsync(
		B.DevPtr(X), B.DevPtr(Y), B.DevPtr(Z),
		m.DevPtr(X), m.DevPtr(Y), m.DevPtr(Z),
		Msat.DevPtr(0), Msat.Mul(0),
		phi.DevPtr(0),
		targetIdx.Ptr,
		sourceIdx.Ptr,
		tensor.DevPtr(0),
		stencilCount,
		targetIdx.Len,
		cfg,
	)
}
