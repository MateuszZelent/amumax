#include "float3.h"
#include "amul.h"
#include "constants.h"

extern "C" __global__ void
add_demag_boundary_corr(float* __restrict__ Bx, float* __restrict__ By, float* __restrict__ Bz,
                        float* __restrict__ mx, float* __restrict__ my, float* __restrict__ mz,
                        float* __restrict__ Ms_, float Ms_mul,
                        float* __restrict__ phi,
                        int* __restrict__ targetIdx,
                        int* __restrict__ sourceIdx,
                        float* __restrict__ tensor,
                        int stencilCount,
                        int nTarget) {

    int t = blockIdx.x * blockDim.x + threadIdx.x;
    if (t >= nTarget) {
        return;
    }

    int I = targetIdx[t];
    if (I < 0) {
        return;
    }

    float phiI = phi[I];
    if (phiI <= 0.0f) {
        return;
    }

    float3 B = make_float3(0.0f, 0.0f, 0.0f);
    int sourceBase = t * stencilCount;
    int tensorBase = sourceBase * 6;

    for (int s = 0; s < stencilCount; s++) {
        int J = sourceIdx[sourceBase + s];
        if (J < 0) {
            continue;
        }

        float phiJ = phi[J];
        if (phiJ <= 0.0f) {
            continue;
        }

        float3 mJ = make_float3(mx[J], my[J], mz[J]);
        if (is0(mJ)) {
            continue;
        }

        float msJ = amul(Ms_, Ms_mul, J);
        if (msJ == 0.0f) {
            continue;
        }

        float3 src = (MU0 * phiJ * msJ) * mJ;
        int k = tensorBase + s * 6;

        float xx = tensor[k + 0];
        float xy = tensor[k + 1];
        float xz = tensor[k + 2];
        float yy = tensor[k + 3];
        float yz = tensor[k + 4];
        float zz = tensor[k + 5];

        B.x += xx * src.x + xy * src.y + xz * src.z;
        B.y += xy * src.x + yy * src.y + yz * src.z;
        B.z += xz * src.x + yz * src.y + zz * src.z;
    }

    Bx[I] += B.x;
    By[I] += B.y;
    Bz[I] += B.z;
}
