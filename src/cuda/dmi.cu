#include <stdint.h>
#include "exchange.h"
#include "float3.h"
#include "stencil.h"
#include "amul.h"

// Exchange + Dzyaloshinskii-Moriya interaction according to
// Bagdanov and Röβler, PRL 87, 3, 2001. eq.8 (out-of-plane symmetry breaking).
// Taking into account proper boundary conditions.
// m: normalized magnetization
// H: effective field in Tesla
// D: dmi strength / Msat, in Tesla*m
// A: Aex/Msat
extern "C" __global__ void
adddmi(float* __restrict__ Hx, float* __restrict__ Hy, float* __restrict__ Hz,
       float* __restrict__ mx, float* __restrict__ my, float* __restrict__ mz,
       float* __restrict__ Ms_, float Ms_mul,
       float* __restrict__ vol,
       float* __restrict__ fxm, float* __restrict__ fxp,
       float* __restrict__ fym, float* __restrict__ fyp,
       float* __restrict__ fzm, float* __restrict__ fzp,
       float* __restrict__ aLUT2d, float* __restrict__ dLUT2d, uint8_t* __restrict__ regions,
       float cx, float cy, float cz, float phiFloor, int Nx, int Ny, int Nz, uint8_t PBC, uint8_t OpenBC) {

    int ix = blockIdx.x * blockDim.x + threadIdx.x;
    int iy = blockIdx.y * blockDim.y + threadIdx.y;
    int iz = blockIdx.z * blockDim.z + threadIdx.z;

    if (ix >= Nx || iy >= Ny || iz >= Nz) {
        return;
    }

    int I = idx(ix, iy, iz);
    float3 h = make_float3(0.0f, 0.0f, 0.0f);
    float3 m0 = make_float3(mx[I], my[I], mz[I]);
    uint8_t r0 = regions[I];
    int i_;

    if (is0(m0)) {
        return;
    }

    float v0 = vol[I];
    if (v0 <= 0.0f) {
        return;
    }
    float invVol = 1.0f / fmaxf(v0, fmaxf(phiFloor, 1e-6f));

    {
        float faceWeight = fxm[I] * invVol;
        if (faceWeight > 0.0f) {
            float3 m1 = make_float3(0.0f, 0.0f, 0.0f);
            int r1 = r0;
            if (ix-1 >= 0 || PBCx) {
                i_ = idx(lclampx(ix-1), iy, iz);
                m1 = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m1)) {
                    r1 = regions[i_];
                }
            }
            float A1 = aLUT2d[symidx(r0, r1)];
            float D1 = dLUT2d[symidx(r0, r1)];
            if (!is0(m1)) {
                h   += faceWeight * ((2.0f*A1/(cx*cx)) * (m1 - m0));
                h.x += faceWeight * (D1/cx) * (-m1.z);
                h.z -= faceWeight * (D1/cx) * (-m1.x);
            } else if (!OpenBC && fxm[I] > 0.999f) {
                m1.x = m0.x - (-cx * (0.5f*D1/A1) * m0.z);
                m1.y = m0.y;
                m1.z = m0.z + (-cx * (0.5f*D1/A1) * m0.x);
                h   += faceWeight * ((2.0f*A1/(cx*cx)) * (m1 - m0));
                h.x += faceWeight * (D1/cx) * (-m1.z);
                h.z -= faceWeight * (D1/cx) * (-m1.x);
            }
        }
    }

    {
        float faceWeight = fxp[I] * invVol;
        if (faceWeight > 0.0f) {
            float3 m2 = make_float3(0.0f, 0.0f, 0.0f);
            int r2 = r0;
            if (ix+1 < Nx || PBCx) {
                i_ = idx(hclampx(ix+1), iy, iz);
                m2 = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m2)) {
                    r2 = regions[i_];
                }
            }
            float A2 = aLUT2d[symidx(r0, r2)];
            float D2 = dLUT2d[symidx(r0, r2)];
            if (!is0(m2)) {
                h   += faceWeight * ((2.0f*A2/(cx*cx)) * (m2 - m0));
                h.x += faceWeight * (D2/cx) * (m2.z);
                h.z -= faceWeight * (D2/cx) * (m2.x);
            } else if (!OpenBC && fxp[I] > 0.999f) {
                m2.x = m0.x - (cx * (0.5f*D2/A2) * m0.z);
                m2.y = m0.y;
                m2.z = m0.z + (cx * (0.5f*D2/A2) * m0.x);
                h   += faceWeight * ((2.0f*A2/(cx*cx)) * (m2 - m0));
                h.x += faceWeight * (D2/cx) * (m2.z);
                h.z -= faceWeight * (D2/cx) * (m2.x);
            }
        }
    }

    {
        float faceWeight = fym[I] * invVol;
        if (faceWeight > 0.0f) {
            float3 m1 = make_float3(0.0f, 0.0f, 0.0f);
            int r1 = r0;
            if (iy-1 >= 0 || PBCy) {
                i_ = idx(ix, lclampy(iy-1), iz);
                m1 = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m1)) {
                    r1 = regions[i_];
                }
            }
            float A1 = aLUT2d[symidx(r0, r1)];
            float D1 = dLUT2d[symidx(r0, r1)];
            if (!is0(m1)) {
                h   += faceWeight * ((2.0f*A1/(cy*cy)) * (m1 - m0));
                h.y += faceWeight * (D1/cy) * (-m1.z);
                h.z -= faceWeight * (D1/cy) * (-m1.y);
            } else if (!OpenBC && fym[I] > 0.999f) {
                m1.x = m0.x;
                m1.y = m0.y - (-cy * (0.5f*D1/A1) * m0.z);
                m1.z = m0.z + (-cy * (0.5f*D1/A1) * m0.y);
                h   += faceWeight * ((2.0f*A1/(cy*cy)) * (m1 - m0));
                h.y += faceWeight * (D1/cy) * (-m1.z);
                h.z -= faceWeight * (D1/cy) * (-m1.y);
            }
        }
    }

    {
        float faceWeight = fyp[I] * invVol;
        if (faceWeight > 0.0f) {
            float3 m2 = make_float3(0.0f, 0.0f, 0.0f);
            int r2 = r0;
            if (iy+1 < Ny || PBCy) {
                i_ = idx(ix, hclampy(iy+1), iz);
                m2 = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m2)) {
                    r2 = regions[i_];
                }
            }
            float A2 = aLUT2d[symidx(r0, r2)];
            float D2 = dLUT2d[symidx(r0, r2)];
            if (!is0(m2)) {
                h   += faceWeight * ((2.0f*A2/(cy*cy)) * (m2 - m0));
                h.y += faceWeight * (D2/cy) * (m2.z);
                h.z -= faceWeight * (D2/cy) * (m2.y);
            } else if (!OpenBC && fyp[I] > 0.999f) {
                m2.x = m0.x;
                m2.y = m0.y - (cy * (0.5f*D2/A2) * m0.z);
                m2.z = m0.z + (cy * (0.5f*D2/A2) * m0.y);
                h   += faceWeight * ((2.0f*A2/(cy*cy)) * (m2 - m0));
                h.y += faceWeight * (D2/cy) * (m2.z);
                h.z -= faceWeight * (D2/cy) * (m2.y);
            }
        }
    }

    if (Nz != 1) {
        {
            float faceWeight = fzm[I] * invVol;
            if (faceWeight > 0.0f) {
                float3 m1 = make_float3(0.0f, 0.0f, 0.0f);
                if (iz-1 >= 0 || PBCz) {
                    i_ = idx(ix, iy, lclampz(iz-1));
                    m1 = make_float3(mx[i_], my[i_], mz[i_]);
                    if (!is0(m1)) {
                        float A1 = aLUT2d[symidx(r0, regions[i_])];
                        h += faceWeight * ((2.0f*A1/(cz*cz)) * (m1 - m0));
                    }
                }
            }
        }

        {
            float faceWeight = fzp[I] * invVol;
            if (faceWeight > 0.0f) {
                float3 m2 = make_float3(0.0f, 0.0f, 0.0f);
                if (iz+1 < Nz || PBCz) {
                    i_ = idx(ix, iy, hclampz(iz+1));
                    m2 = make_float3(mx[i_], my[i_], mz[i_]);
                    if (!is0(m2)) {
                        float A2 = aLUT2d[symidx(r0, regions[i_])];
                        h += faceWeight * ((2.0f*A2/(cz*cz)) * (m2 - m0));
                    }
                }
            }
        }
    }

    float invMs = inv_Msat(Ms_, Ms_mul, I);
    Hx[I] += h.x * invMs;
    Hy[I] += h.y * invMs;
    Hz[I] += h.z * invMs;
}

// Note on boundary conditions.
//
// We need the derivative and laplacian of m in point A, but e.g. C lies out of the boundaries.
// We use the boundary condition in B (derivative of the magnetization) to extrapolate m to point C:
// 	m_C = m_A + (dm/dx)|_B * cellsize
//
// When point C is inside the boundary, we just use its actual value.
//
// Then we can take the central derivative in A:
// 	(dm/dx)|_A = (m_C - m_D) / (2*cellsize)
// And the laplacian:
// 	lapl(m)|_A = (m_C + m_D - 2*m_A) / (cellsize^2)
//
// All these operations should be second order as they involve only central derivatives.
//
//    ------------------------------------------------------------------ *
//   |                                                   |             C |
//   |                                                   |          **   |
//   |                                                   |        ***    |
//   |                                                   |     ***       |
//   |                                                   |   ***         |
//   |                                                   | ***           |
//   |                                                   B               |
//   |                                               *** |               |
//   |                                            ***    |               |
//   |                                         ****      |               |
//   |                                     ****          |               |
//   |                                  ****             |               |
//   |                              ** A                 |               |
//   |                         *****                     |               |
//   |                   ******                          |               |
//   |          *********                                |               |
//   |D ********                                         |               |
//   |                                                   |               |
//   +----------------+----------------+-----------------+---------------+
//  -1              -0.5               0               0.5               1
//                                 x
