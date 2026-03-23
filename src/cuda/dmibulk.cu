#include <stdint.h>
#include <math.h>
#include "exchange.h"
#include "float3.h"
#include "stencil.h"
#include "amul.h"

inline __device__ float3 axis_boundary_normal(int axis, bool positive) {
    switch (axis) {
        case 0: return positive ? make_float3( 1.0f, 0.0f, 0.0f) : make_float3(-1.0f, 0.0f, 0.0f);
        case 1: return positive ? make_float3( 0.0f, 1.0f, 0.0f) : make_float3( 0.0f,-1.0f, 0.0f);
        default: return positive ? make_float3( 0.0f, 0.0f, 1.0f) : make_float3( 0.0f, 0.0f,-1.0f);
    }
}

inline __device__ float3 cutcell_boundary_normal(float fxm, float fxp, float fym, float fyp, float fzm, float fzp,
                                                 bool xmExposed, bool xpExposed,
                                                 bool ymExposed, bool ypExposed,
                                                 bool zmExposed, bool zpExposed,
                                                 int axis, bool positive) {
    float3 raw = make_float3(
        (xpExposed ? fxp : 0.0f) - (xmExposed ? fxm : 0.0f),
        (ypExposed ? fyp : 0.0f) - (ymExposed ? fym : 0.0f),
        (zpExposed ? fzp : 0.0f) - (zmExposed ? fzm : 0.0f));
    float nlen = len(raw);
    if (nlen <= 1e-12f) {
        return axis_boundary_normal(axis, positive);
    }
    return (1.0f / nlen) * raw;
}

inline __device__ float3 bulk_boundary_derivative(float3 m, float3 n, float D_2A) {
    return D_2A * cross(n, m);
}

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

    float fxmI = fxm[I], fxpI = fxp[I];
    float fymI = fym[I], fypI = fyp[I];
    float fzmI = fzm[I], fzpI = fzp[I];

    float3 mxm = make_float3(0.0f, 0.0f, 0.0f), mxp = make_float3(0.0f, 0.0f, 0.0f);
    float3 mym = make_float3(0.0f, 0.0f, 0.0f), myp = make_float3(0.0f, 0.0f, 0.0f);
    float3 mzm = make_float3(0.0f, 0.0f, 0.0f), mzp = make_float3(0.0f, 0.0f, 0.0f);
    bool hasXm = false, hasXp = false, hasYm = false, hasYp = false, hasZm = false, hasZp = false;
    int rXm = r0, rXp = r0, rYm = r0, rYp = r0, rZm = r0, rZp = r0;

    if (ix-1 >= 0 || PBCx) {
        i_ = idx(lclampx(ix-1), iy, iz);
        mxm = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(mxm)) {
            hasXm = true;
            rXm = regions[i_];
        }
    }
    if (ix+1 < Nx || PBCx) {
        i_ = idx(hclampx(ix+1), iy, iz);
        mxp = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(mxp)) {
            hasXp = true;
            rXp = regions[i_];
        }
    }
    if (iy-1 >= 0 || PBCy) {
        i_ = idx(ix, lclampy(iy-1), iz);
        mym = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(mym)) {
            hasYm = true;
            rYm = regions[i_];
        }
    }
    if (iy+1 < Ny || PBCy) {
        i_ = idx(ix, hclampy(iy+1), iz);
        myp = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(myp)) {
            hasYp = true;
            rYp = regions[i_];
        }
    }
    if (iz-1 >= 0 || PBCz) {
        i_ = idx(ix, iy, lclampz(iz-1));
        mzm = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(mzm)) {
            hasZm = true;
            rZm = regions[i_];
        }
    }
    if (iz+1 < Nz || PBCz) {
        i_ = idx(ix, iy, hclampz(iz+1));
        mzp = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(mzp)) {
            hasZp = true;
            rZp = regions[i_];
        }
    }

    bool xmExposed = fxmI > 0.0f && !hasXm;
    bool xpExposed = fxpI > 0.0f && !hasXp;
    bool ymExposed = fymI > 0.0f && !hasYm;
    bool ypExposed = fypI > 0.0f && !hasYp;
    bool zmExposed = fzmI > 0.0f && !hasZm;
    bool zpExposed = fzpI > 0.0f && !hasZp;

    {
        float faceWeight = fxmI * invVol;
        if (faceWeight > 0.0f) {
            float3 m1 = mxm;
            int r1 = rXm;
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = (fabsf(A) > 1e-30f) ? (D / (2.0f*A)) : 0.0f;
            if (hasXm) {
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m1 - m0));
                h.y += faceWeight * (D/cx) * (-m1.z);
                h.z -= faceWeight * (D/cx) * (-m1.y);
            } else if (!OpenBC && fxmI > 0.0f && D_2A != 0.0f) {
                float3 n = cutcell_boundary_normal(fxmI, fxpI, fymI, fypI, fzmI, fzpI, xmExposed, xpExposed, ymExposed, ypExposed, zmExposed, zpExposed, 0, false);
                m1 = m0 + cx * bulk_boundary_derivative(m0, n, D_2A);
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m1 - m0));
                h.y += faceWeight * (D/cx) * (-m1.z);
                h.z -= faceWeight * (D/cx) * (-m1.y);
            }
        }
    }

    {
        float faceWeight = fxpI * invVol;
        if (faceWeight > 0.0f) {
            float3 m2 = mxp;
            int r1 = rXp;
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = (fabsf(A) > 1e-30f) ? (D / (2.0f*A)) : 0.0f;
            if (hasXp) {
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m2 - m0));
                h.y += faceWeight * (D/cx) * (m2.z);
                h.z -= faceWeight * (D/cx) * (m2.y);
            } else if (!OpenBC && fxpI > 0.0f && D_2A != 0.0f) {
                float3 n = cutcell_boundary_normal(fxmI, fxpI, fymI, fypI, fzmI, fzpI, xmExposed, xpExposed, ymExposed, ypExposed, zmExposed, zpExposed, 0, true);
                m2 = m0 + cx * bulk_boundary_derivative(m0, n, D_2A);
                h   += faceWeight * ((2.0f*A/(cx*cx)) * (m2 - m0));
                h.y += faceWeight * (D/cx) * (m2.z);
                h.z -= faceWeight * (D/cx) * (m2.y);
            }
        }
    }

    {
        float faceWeight = fymI * invVol;
        if (faceWeight > 0.0f) {
            float3 m1 = mym;
            int r1 = rYm;
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = (fabsf(A) > 1e-30f) ? (D / (2.0f*A)) : 0.0f;
            if (hasYm) {
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m1 - m0));
                h.x -= faceWeight * (D/cy) * (-m1.z);
                h.z += faceWeight * (D/cy) * (-m1.x);
            } else if (!OpenBC && fymI > 0.0f && D_2A != 0.0f) {
                float3 n = cutcell_boundary_normal(fxmI, fxpI, fymI, fypI, fzmI, fzpI, xmExposed, xpExposed, ymExposed, ypExposed, zmExposed, zpExposed, 1, false);
                m1 = m0 + cy * bulk_boundary_derivative(m0, n, D_2A);
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m1 - m0));
                h.x -= faceWeight * (D/cy) * (-m1.z);
                h.z += faceWeight * (D/cy) * (-m1.x);
            }
        }
    }

    {
        float faceWeight = fypI * invVol;
        if (faceWeight > 0.0f) {
            float3 m2 = myp;
            int r1 = rYp;
            float A = aLUT2d[symidx(r0, r1)];
            float D = DLUT2d[symidx(r0, r1)];
            float D_2A = (fabsf(A) > 1e-30f) ? (D / (2.0f*A)) : 0.0f;
            if (hasYp) {
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m2 - m0));
                h.x -= faceWeight * (D/cy) * (m2.z);
                h.z += faceWeight * (D/cy) * (m2.x);
            } else if (!OpenBC && fypI > 0.0f && D_2A != 0.0f) {
                float3 n = cutcell_boundary_normal(fxmI, fxpI, fymI, fypI, fzmI, fzpI, xmExposed, xpExposed, ymExposed, ypExposed, zmExposed, zpExposed, 1, true);
                m2 = m0 + cy * bulk_boundary_derivative(m0, n, D_2A);
                h   += faceWeight * ((2.0f*A/(cy*cy)) * (m2 - m0));
                h.x -= faceWeight * (D/cy) * (m2.z);
                h.z += faceWeight * (D/cy) * (m2.x);
            }
        }
    }

    if ((Nz != 1) || (!OpenBC)) {
        {
            float faceWeight = fzmI * invVol;
            if (faceWeight > 0.0f) {
                float3 m1 = mzm;
                int r1 = rZm;
                float A = aLUT2d[symidx(r0, r1)];
                float D = DLUT2d[symidx(r0, r1)];
                float D_2A = (fabsf(A) > 1e-30f) ? (D / (2.0f*A)) : 0.0f;
                if (hasZm) {
                    h   += faceWeight * ((2.0f*A/(cz*cz)) * (m1 - m0));
                    h.x += faceWeight * (D/cz) * (-m1.y);
                    h.y -= faceWeight * (D/cz) * (-m1.x);
                } else if (!OpenBC && fzmI > 0.0f && D_2A != 0.0f) {
                    float3 n = cutcell_boundary_normal(fxmI, fxpI, fymI, fypI, fzmI, fzpI, xmExposed, xpExposed, ymExposed, ypExposed, zmExposed, zpExposed, 2, false);
                    m1 = m0 + cz * bulk_boundary_derivative(m0, n, D_2A);
                    h   += faceWeight * ((2.0f*A/(cz*cz)) * (m1 - m0));
                    h.x += faceWeight * (D/cz) * (-m1.y);
                    h.y -= faceWeight * (D/cz) * (-m1.x);
                }
            }
        }

        {
            float faceWeight = fzpI * invVol;
            if (faceWeight > 0.0f) {
                float3 m2 = mzp;
                int r1 = rZp;
                float A = aLUT2d[symidx(r0, r1)];
                float D = DLUT2d[symidx(r0, r1)];
                float D_2A = (fabsf(A) > 1e-30f) ? (D / (2.0f*A)) : 0.0f;
                if (hasZp) {
                    h   += faceWeight * ((2.0f*A/(cz*cz)) * (m2 - m0));
                    h.x += faceWeight * (D/cz) * (m2.y);
                    h.y -= faceWeight * (D/cz) * (m2.x);
                } else if (!OpenBC && fzpI > 0.0f && D_2A != 0.0f) {
                    float3 n = cutcell_boundary_normal(fxmI, fxpI, fymI, fypI, fzmI, fzpI, xmExposed, xpExposed, ymExposed, ypExposed, zmExposed, zpExposed, 2, true);
                    m2 = m0 + cz * bulk_boundary_derivative(m0, n, D_2A);
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
