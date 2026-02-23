#include "stencil.h"

// Copy src (size S, smaller) to dst (size D, larger).
// dst must be pre-zeroed (for zero-padding).
// Optionally multiplies by vol (geometry mask). vol may be NULL.
extern "C" __global__ void
copypad(float* __restrict__ dst, int Dx, int Dy, int Dz,
        float* __restrict__ src, int Sx, int Sy, int Sz,
        float* __restrict__ vol) {

    int ix = blockIdx.x * blockDim.x + threadIdx.x;
    int iy = blockIdx.y * blockDim.y + threadIdx.y;
    int iz = blockIdx.z * blockDim.z + threadIdx.z;

    if (ix < Sx && iy < Sy && iz < Sz) {
        int sI = index(ix, iy, iz, Sx, Sy, Sz);
        float v = (vol == NULL) ? 1.0f : vol[sI];
        dst[index(ix, iy, iz, Dx, Dy, Dz)] = v * src[sI];
    }
}
