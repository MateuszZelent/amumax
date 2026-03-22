#include <stdint.h>
#include "exchange.h"
#include "float3.h"
#include "stencil.h"
#include "amul.h"

// See exchange.go for more details.
extern "C" __global__ void
addexchange(float* __restrict__ Bx, float* __restrict__ By, float* __restrict__ Bz,
            float* __restrict__ mx, float* __restrict__ my, float* __restrict__ mz,
            float* __restrict__ Ms_, float Ms_mul,
            float* __restrict__ vol,
            float* __restrict__ fxm, float* __restrict__ fxp,
            float* __restrict__ fym, float* __restrict__ fyp,
            float* __restrict__ fzm, float* __restrict__ fzp,
            float* __restrict__ aLUT2d, uint8_t* __restrict__ regions,
            float wx, float wy, float wz, int Nx, int Ny, int Nz, uint8_t PBC) {

    int ix = blockIdx.x * blockDim.x + threadIdx.x;
    int iy = blockIdx.y * blockDim.y + threadIdx.y;
    int iz = blockIdx.z * blockDim.z + threadIdx.z;

    if (ix >= Nx || iy >= Ny || iz >= Nz) {
        return;
    }

    int I = idx(ix, iy, iz);
    float3 m0 = make_float3(mx[I], my[I], mz[I]);
    if (is0(m0)) {
        return;
    }

    float v0 = vol[I];
    if (v0 <= 0.0f) {
        return;
    }
    float invVol = 1.0f / fmaxf(v0, 1e-6f);

    uint8_t r0 = regions[I];
    float3 B = make_float3(0.0f, 0.0f, 0.0f);

    int i_;
    float3 m_;
    float a__;
    float faceWeight;

    if (ix-1 >= 0 || PBCx) {
        faceWeight = fxm[I] * invVol;
        if (faceWeight > 0.0f) {
            i_ = idx(lclampx(ix-1), iy, iz);
            m_ = make_float3(mx[i_], my[i_], mz[i_]);
            if (!is0(m_)) {
                a__ = aLUT2d[symidx(r0, regions[i_])];
                B += (faceWeight * wx * a__) * (m_ - m0);
            }
        }
    }

    if (ix+1 < Nx || PBCx) {
        faceWeight = fxp[I] * invVol;
        if (faceWeight > 0.0f) {
            i_ = idx(hclampx(ix+1), iy, iz);
            m_ = make_float3(mx[i_], my[i_], mz[i_]);
            if (!is0(m_)) {
                a__ = aLUT2d[symidx(r0, regions[i_])];
                B += (faceWeight * wx * a__) * (m_ - m0);
            }
        }
    }

    if (iy-1 >= 0 || PBCy) {
        faceWeight = fym[I] * invVol;
        if (faceWeight > 0.0f) {
            i_ = idx(ix, lclampy(iy-1), iz);
            m_ = make_float3(mx[i_], my[i_], mz[i_]);
            if (!is0(m_)) {
                a__ = aLUT2d[symidx(r0, regions[i_])];
                B += (faceWeight * wy * a__) * (m_ - m0);
            }
        }
    }

    if (iy+1 < Ny || PBCy) {
        faceWeight = fyp[I] * invVol;
        if (faceWeight > 0.0f) {
            i_ = idx(ix, hclampy(iy+1), iz);
            m_ = make_float3(mx[i_], my[i_], mz[i_]);
            if (!is0(m_)) {
                a__ = aLUT2d[symidx(r0, regions[i_])];
                B += (faceWeight * wy * a__) * (m_ - m0);
            }
        }
    }

    if (Nz != 1) {
        if (iz-1 >= 0 || PBCz) {
            faceWeight = fzm[I] * invVol;
            if (faceWeight > 0.0f) {
                i_ = idx(ix, iy, lclampz(iz-1));
                m_ = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m_)) {
                    a__ = aLUT2d[symidx(r0, regions[i_])];
                    B += (faceWeight * wz * a__) * (m_ - m0);
                }
            }
        }

        if (iz+1 < Nz || PBCz) {
            faceWeight = fzp[I] * invVol;
            if (faceWeight > 0.0f) {
                i_ = idx(ix, iy, hclampz(iz+1));
                m_ = make_float3(mx[i_], my[i_], mz[i_]);
                if (!is0(m_)) {
                    a__ = aLUT2d[symidx(r0, regions[i_])];
                    B += (faceWeight * wz * a__) * (m_ - m0);
                }
            }
        }
    }

    float invMs = inv_Msat(Ms_, Ms_mul, I);
    Bx[I] += B.x * invMs;
    By[I] += B.y * invMs;
    Bz[I] += B.z * invMs;
}
