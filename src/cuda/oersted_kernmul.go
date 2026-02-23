package cuda

// Oersted field: cross-product kernel multiplication in Fourier space.
// Computes B = J × K (in-place, overwrites fftJ) where both J and K
// have already been FFT'd.

import (
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
)

// OerstedKernmul3DAsync performs the cross-product multiplication
// B = J × K in Fourier space, writing the result back into fftJ.
//
// fftJ[3]: FFT'd current density components (overwritten with result).
// fftK[3]: FFT'd Biot-Savart kernel components (read-only).
// Nx, Ny, Nz: logic sizes in complex elements.
func OerstedKernmul3DAsync(fftJ, fftK [3]*data.Slice, Nx, Ny, Nz int) {
	log.AssertMsg(fftJ[X].NComp() == 1 && fftK[X].NComp() == 1,
		"OerstedKernmul3DAsync: slices must have NComp()==1")

	cfg := make3DConf([3]int{Nx, Ny, Nz})
	kOerstedkernmul3dAsync(
		fftJ[X].DevPtr(0), fftJ[Y].DevPtr(0), fftJ[Z].DevPtr(0),
		fftK[X].DevPtr(0), fftK[Y].DevPtr(0), fftK[Z].DevPtr(0),
		Nx, Ny, Nz, cfg)
}
