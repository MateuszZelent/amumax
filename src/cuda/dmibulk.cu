#include <stdint.h>
#include "exchange.h"
#include "float3.h"
#include "stencil.h"
#include "amul.h"

// Exchange + Dzyaloshinskii-Moriya interaction for bulk material.
// Energy:
//
// 	E  = D M . rot(M)
//
// Effective field:
//
// 	Hx = 2A/Bs nabla²Mx + 2D/Bs dzMy - 2D/Bs dyMz
// 	Hy = 2A/Bs nabla²My + 2D/Bs dxMz - 2D/Bs dzMx
// 	Hz = 2A/Bs nabla²Mz + 2D/Bs dyMx - 2D/Bs dxMy
//
// Boundary conditions:
//
// 	        2A dxMx = 0
// 	 D Mz + 2A dxMy = 0
// 	-D My + 2A dxMz = 0
//
// 	-D Mz + 2A dyMx = 0
// 	        2A dyMy = 0
// 	 D Mx + 2A dyMz = 0
//
// 	 D My + 2A dzMx = 0
// 	-D Mx + 2A dzMy = 0
// 	        2A dzMz = 0
//
extern "C" __global__ void
adddmibulk(float* __restrict__ Hx, float* __restrict__ Hy, float* __restrict__ Hz,
           float* __restrict__ mx, float* __restrict__ my, float* __restrict__ mz,
           float* __restrict__ Ms_, float Ms_mul,
           float* __restrict__ vol,
           float* __restrict__ fxm, float* __restrict__ fxp,
           float* __restrict__ fym, float* __restrict__ fyp,
           float* __restrict__ fzm, float* __restrict__ fzp,
           float* __restrict__ aLUT2d, float* __restrict__ DLUT2d,
           uint8_t* __restrict__ regions,
           float cx, float cy, float cz, int Nx, int Ny, int Nz, uint8_t PBC, uint8_t OpenBC) {

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
    float invVol = 1.0f / fmaxf(v0, 1e-6f);

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
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = D / (2.0f*A);
            if (!is0(m1)) {
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m1 - m0));
                h.y += faceWeight * (D/cx) * (-m1.z);
                h.z -= faceWeight * (D/cx) * (-m1.y);
            } else if (!OpenBC) {
                m1.x = m0.x;
                m1.y = m0.y - (-cx * D_2A * m0.z);
                m1.z = m0.z + (-cx * D_2A * m0.y);
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m1 - m0));
                h.y += faceWeight * (D/cx) * (-m1.z);
                h.z -= faceWeight * (D/cx) * (-m1.y);
            }
        }
    }

    {
        float faceWeight = fxp[I] * invVol;
        if (faceWeight > 0.0f) {
            float3 m2 = make_float3(0.0f, 0.0f, 0.0f);
            int r1 = r0;
            if (ix+1 < Nx || PBCx) {
                i_ = idx(hclampx(ix+1), iy, iz);
                m2 = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m2)) {
                    r1 = regions[i_];
                }
            }
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = D / (2.0f*A);
            if (!is0(m2)) {
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m2 - m0));
                h.y += faceWeight * (D/cx) * (m2.z);
                h.z -= faceWeight * (D/cx) * (m2.y);
            } else if (!OpenBC) {
                m2.x = m0.x;
                m2.y = m0.y - (+cx * D_2A * m0.z);
                m2.z = m0.z + (+cx * D_2A * m0.y);
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m2 - m0));
                h.y += faceWeight * (D/cx) * (m2.z);
                h.z -= faceWeight * (D/cx) * (m2.y);
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
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = D / (2.0f*A);
            if (!is0(m1)) {
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m1 - m0));
                h.x -= faceWeight * (D/cy) * (-m1.z);
                h.z += faceWeight * (D/cy) * (-m1.x);
            } else if (!OpenBC) {
                m1.x = m0.x + (-cy * D_2A * m0.z);
                m1.y = m0.y;
                m1.z = m0.z - (-cy * D_2A * m0.x);
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m1 - m0));
                h.x -= faceWeight * (D/cy) * (-m1.z);
                h.z += faceWeight * (D/cy) * (-m1.x);
            }
        }
    }

    {
        float faceWeight = fyp[I] * invVol;
        if (faceWeight > 0.0f) {
            float3 m2 = make_float3(0.0f, 0.0f, 0.0f);
            int r1 = r0;
            if (iy+1 < Ny || PBCy) {
                i_ = idx(ix, hclampy(iy+1), iz);
                m2 = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m2)) {
                    r1 = regions[i_];
                }
            }
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = D / (2.0f*A);
            if (!is0(m2)) {
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m2 - m0));
                h.x -= faceWeight * (D/cy) * (m2.z);
                h.z += faceWeight * (D/cy) * (m2.x);
            } else if (!OpenBC) {
                m2.x = m0.x + (+cy * D_2A * m0.z);
                m2.y = m0.y;
                m2.z = m0.z - (+cy * D_2A * m0.x);
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m2 - m0));
                h.x -= faceWeight * (D/cy) * (m2.z);
                h.z += faceWeight * (D/cy) * (m2.x);
            }
        }
    }

    if ((Nz != 1) || (!OpenBC)) {
        {
            float faceWeight = fzm[I] * invVol;
            if (faceWeight > 0.0f) {
                float3 m1 = make_float3(0.0f, 0.0f, 0.0f);
                int r1 = r0;
                if (iz-1 >= 0 || PBCz) {
                    i_ = idx(ix, iy, lclampz(iz-1));
                    m1 = make_float3(mx[i_], my[i_], mz[i_]);
                    if (!is0(m1)) {
                        r1 = regions[i_];
                    }
                }
                float A = aLUT2d[symidx(r0, r1)];
                float D = DLUT2d[symidx(r0, r1)];
                float D_2A = D / (2.0f*A);
                if (!is0(m1)) {
                    h   += faceWeight * ((2.0f*A/(cz*cz)) * (m1 - m0));
                    h.x += faceWeight * (D/cz) * (-m1.y);
                    h.y -= faceWeight * (D/cz) * (-m1.x);
                } else if (!OpenBC) {
                    m1.x = m0.x - (-cz * D_2A * m0.y);
                    m1.y = m0.y + (-cz * D_2A * m0.x);
                    m1.z = m0.z;
                    h   += faceWeight * ((2.0f*A/(cz*cz)) * (m1 - m0));
                    h.x += faceWeight * (D/cz) * (-m1.y);
                    h.y -= faceWeight * (D/cz) * (-m1.x);
                }
            }
        }

        {
            float faceWeight = fzp[I] * invVol;
            if (faceWeight > 0.0f) {
                float3 m2 = make_float3(0.0f, 0.0f, 0.0f);
                int r1 = r0;
                if (iz+1 < Nz || PBCz) {
                    i_ = idx(ix, iy, hclampz(iz+1));
                    m2 = make_float3(mx[i_], my[i_], mz[i_]);
                    if (!is0(m2)) {
                        r1 = regions[i_];
                    }
                }
                float A = aLUT2d[symidx(r0, r1)];
                float D = DLUT2d[symidx(r0, r1)];
                float D_2A = D / (2.0f*A);
                if (!is0(m2)) {
                    h   += faceWeight * ((2.0f*A/(cz*cz)) * (m2 - m0));
                    h.x += faceWeight * (D/cz) * (m2.y);
                    h.y -= faceWeight * (D/cz) * (m2.x);
                } else if (!OpenBC) {
                    m2.x = m0.x - (+cz * D_2A * m0.y);
                    m2.y = m0.y + (+cz * D_2A * m0.x);
                    m2.z = m0.z;
                    h   += faceWeight * ((2.0f*A/(cz*cz)) * (m2 - m0));
                    h.x += faceWeight * (D/cz) * (m2.y);
                    h.y -= faceWeight * (D/cz) * (m2.x);
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
