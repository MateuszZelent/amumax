package cuda

import (
	"unsafe"

	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
)

// OerstedConvolution performs FFT-accelerated convolution for the Oersted
// field using a pre-computed real-space Biot-Savart kernel (like demag).
//
// The kernel K(r) = mu0/(4pi) * dV * r / |r|^3 is FFT'd once at init time
// and stored on the GPU.  At runtime the cross-product multiplication
// B = J × K is carried out in Fourier space.
type OerstedConvolution struct {
	inputSize        [3]int         // 3D size of the input/output data
	realKernSize     [3]int         // Size of zero-padded FFT domain
	fftKernLogicSize [3]int         // logic size of FFT output in complex elements
	fftRBuf          [3]*data.Slice // FFT real-space work buffers (padded)
	fftCBuf          [3]*data.Slice // FFT complex output buffers
	kern             [3]*data.Slice // FFT'd kernel on GPU (full complex, Kx/Ky/Kz)
	fwPlan           fft3DR2CPlan   // Forward FFT (1 component)
	bwPlan           fft3DC2RPlan   // Backward FFT (1 component)
}

// NewOersted creates a new OerstedConvolution.
//
// kernel must be the [3]*data.Slice returned by mag.OerstedKernel — three
// single-component slices of identical padded size.  The constructor copies
// the kernel to the GPU, FFTs each component, scales by 1/N (FFT
// normalisation), and keeps the full complex result.  The host-side kernel
// can be freed by the caller afterwards.
func NewOersted(inputSize [3]int, kernel [3]*data.Slice) *OerstedConvolution {
	c := new(OerstedConvolution)
	c.inputSize = inputSize
	c.realKernSize = kernel[0].Size()
	c.init(kernel)
	return c
}

// Exec computes the Oersted field B from current density J, masked by vol.
//
//	B:   output Oersted field (3-comp, inputSize)
//	J:   input current density (3-comp, inputSize) in A/m²
//	vol: geometry mask (1-comp, inputSize), may be nil
func (c *OerstedConvolution) Exec(B, J, vol *data.Slice) {
	log.AssertMsg(B.Size() == c.inputSize && J.Size() == c.inputSize,
		"OerstedConvolution.Exec: size mismatch")

	// Forward FFT all 3 components of J
	for i := 0; i < 3; i++ {
		c.fwFFT(i, J, vol)
	}

	// Cross-product multiplication in Fourier space: B = J × K
	Nx := c.fftKernLogicSize[X]
	Ny := c.fftKernLogicSize[Y]
	Nz := c.fftKernLogicSize[Z]
	OerstedKernmul3DAsync(c.fftCBuf, c.kern, Nx, Ny, Nz)

	// Backward FFT and unpad
	for i := 0; i < 3; i++ {
		c.bwFFT(i, B)
	}
}

// forward FFT component i of J, masked by vol.
// Uses the simple copypad kernel (no Msat multiplication).
func (c *OerstedConvolution) fwFFT(i int, J, vol *data.Slice) {
	zero1Async(c.fftRBuf[i])
	in := J.Comp(i)
	copyPadVol(c.fftRBuf[i], in, vol, c.realKernSize, c.inputSize)
	c.fwPlan.ExecAsync(c.fftRBuf[i], c.fftCBuf[i])
}

// backward FFT component i and copy-unpad result to output
func (c *OerstedConvolution) bwFFT(i int, outp *data.Slice) {
	c.bwPlan.ExecAsync(c.fftCBuf[i], c.fftRBuf[i])
	out := outp.Comp(i)
	copyUnPad(out, c.fftRBuf[i], c.inputSize, c.realKernSize)
}

func (c *OerstedConvolution) init(realKern [3]*data.Slice) {
	nc := fftR2COutputSizeFloats(c.realKernSize)

	// Allocate FFT work buffers — for the cross-product kernel all 3
	// components are needed simultaneously, even in the 2-D case.
	c.fftCBuf[X] = NewSlice(1, nc)
	c.fftCBuf[Y] = NewSlice(1, nc)
	c.fftCBuf[Z] = NewSlice(1, nc)

	c.fftRBuf[X] = NewSlice(1, c.realKernSize)
	c.fftRBuf[Y] = NewSlice(1, c.realKernSize)
	c.fftRBuf[Z] = NewSlice(1, c.realKernSize)

	// FFT plans
	c.fwPlan = newFFT3DR2C(c.realKernSize[X], c.realKernSize[Y], c.realKernSize[Z])
	c.bwPlan = newFFT3DC2R(c.realKernSize[X], c.realKernSize[Y], c.realKernSize[Z])

	// Logic size of FFT output in complex elements
	c.fftKernLogicSize = fftR2COutputSizeFloats(c.realKernSize)
	log.AssertMsg(c.fftKernLogicSize[X]%2 == 0,
		"fftKernLogicSize[X] must be even in OerstedConvolution.init")
	c.fftKernLogicSize[X] /= 2 // complex count

	// FFT the three kernel components, scale by 1/N, and store on GPU.
	// We keep full complex data (no symmetry exploitation — the Oersted
	// kernel is antisymmetric, so its FFT is purely imaginary, but that
	// optimisation can be added later).
	scale := float32(1.0) / float32(c.fwPlan.InputLen())
	output := c.fftCBuf[0]             // reuse work buffer for the FFT
	input := c.fftRBuf[0]              // reuse work buffer
	kfull := data.NewSlice(1, nc)      // host copy of full FFT output
	kfulls := kfull.Host()[0]

	for i := 0; i < 3; i++ {
		if realKern[i] == nil {
			continue
		}
		// Upload real-space kernel to GPU, FFT it
		data.Copy(input, realKern[i])
		c.fwPlan.ExecAsync(input, output)

		// Download full FFT result to host
		data.Copy(kfull, output)

		// Scale every float (re and im) by 1/N
		for j := range kfulls {
			kfulls[j] *= scale
		}

		// Upload scaled kernel to GPU
		c.kern[i] = GPUCopy(kfull)
	}

	kfull.Free()
}

// Free releases all GPU resources.
func (c *OerstedConvolution) Free() {
	if c == nil {
		return
	}
	c.inputSize = [3]int{}
	c.realKernSize = [3]int{}

	for i := 0; i < 3; i++ {
		c.fftCBuf[i].Free()
		c.fftRBuf[i].Free()
		c.fftCBuf[i] = nil
		c.fftRBuf[i] = nil
		c.kern[i].Free()
		c.kern[i] = nil
	}
	c.fwPlan.Free()
	c.bwPlan.Free()

	cudaCtx.SetCurrent()
}

// copyPadVol copies src into the larger dst, multiplying by vol (geometry
// mask).  dst must be pre-zeroed.  vol may be nil.
func copyPadVol(dst, src, vol *data.Slice, dstSize, srcSize [3]int) {
	log.AssertMsg(dst.NComp() == 1 && src.NComp() == 1,
		"copyPadVol: dst and src must have NComp()==1")
	log.AssertMsg(dst.Len() == prod(dstSize) && src.Len() == prod(srcSize),
		"copyPadVol: length mismatch")

	var volPtr unsafe.Pointer
	if vol != nil {
		volPtr = vol.DevPtr(0)
	}

	cfg := make3DConf(srcSize)
	kCopypadAsync(dst.DevPtr(0), dstSize[X], dstSize[Y], dstSize[Z],
		src.DevPtr(0), srcSize[X], srcSize[Y], srcSize[Z],
		volPtr, cfg)
}
