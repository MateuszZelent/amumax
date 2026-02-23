package mag

import (
	"math"

	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
)

// OerstedKernel returns the real-space Biot-Savart kernel components [Kx, Ky, Kz]
// on the padded grid (same padding logic as DemagKernel).
//
// The kernel encodes the vector field:
//
//	K(r) = (mu0 / 4pi) * dV * r / |r|^3
//
// where dV is the cell volume and r is the displacement vector.
// The Oersted field is then obtained by cross-product convolution:
//
//	B = J × K  (convolution)
//
// i.e. Bx = Jy*Kz - Jz*Ky, etc.
//
// The kernel is computed on a "wrap-around" grid identical to demag:
// indices are wrapped so that the kernel naturally represents the
// finite-range Green's function with correct zero-padding for OBC.
//
// PBC is not supported; the function panics if any pbc[i] != 0.
func OerstedKernel(gridsize [3]int, cellsize [3]float64, pbc [3]int) [3]*data.Slice {
	// PBC not supported for Oersted field yet
	for i := range 3 {
		if pbc[i] != 0 {
			log.Log.ErrAndExit("Oersted field does not support periodic boundary conditions (PBC != 0 in direction %d)", i)
		}
	}

	// Padded size (same as demag)
	size := padSize(gridsize, pbc)

	log.AssertMsg(size[Z] > 0 && size[Y] > 0 && size[X] > 0,
		"OerstedKernel: grid size dimensions must be > 0")
	log.AssertMsg(cellsize[X] > 0 && cellsize[Y] > 0 && cellsize[Z] > 0,
		"OerstedKernel: cell size dimensions must be positive")

	// Allocate kernel components
	var kernel [3]*data.Slice
	for i := range 3 {
		kernel[i] = data.NewSlice(1, size)
	}
	arrayKx := kernel[X].Scalars()
	arrayKy := kernel[Y].Scalars()
	arrayKz := kernel[Z].Scalars()

	// Integration ranges (same as demag)
	r1, r2 := kernelRanges(size, pbc)

	// Physical constants
	mu0_over_4pi := 1e-7 // T·m/A (= mu0 / (4*pi))
	dV := cellsize[X] * cellsize[Y] * cellsize[Z]
	prefactor := mu0_over_4pi * dV

	for iz := r1[Z]; iz <= r2[Z]; iz++ {
		zw := wrap(iz, size[Z])
		rz := float64(iz) * cellsize[Z]

		for iy := r1[Y]; iy <= r2[Y]; iy++ {
			yw := wrap(iy, size[Y])
			ry := float64(iy) * cellsize[Y]

			for ix := r1[X]; ix <= r2[X]; ix++ {
				xw := wrap(ix, size[X])
				rx := float64(ix) * cellsize[X]

				r2val := rx*rx + ry*ry + rz*rz
				if r2val == 0 {
					// Self-term: set to zero (no self-field from Biot-Savart)
					continue
				}

				r := math.Sqrt(r2val)
				r3 := r * r2val // |r|^3
				scale := prefactor / r3

				// K(r) = prefactor * r / |r|^3
				// += needed: wrap can alias multiple displacements to the same cell
				arrayKx[zw][yw][xw] += float32(scale * rx)
				arrayKy[zw][yw][xw] += float32(scale * ry)
				arrayKz[zw][yw][xw] += float32(scale * rz)
			}
		}
	}

	return kernel
}
