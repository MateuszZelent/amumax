#include <stdint.h>
#include "exchange.h"
#include "float3.h"
#include "stencil.h"
#include "amul.h"

__device__ inline float u8frac(uint8_t v) {
    return ((float)v) * (1.0f / 255.0f);
}

__device__ inline float link_xm(const uint8_t* __restrict__ linkX, int ix, int iy, int iz, int Nx, int Ny, int Nz, uint8_t PBC) {
    if (ix > 0) {
        return u8frac(linkX[idx(ix - 1, iy, iz)]);
    }
    return PBCx ? u8frac(linkX[idx(Nx - 1, iy, iz)]) : 0.0f;
}

__device__ inline float link_xp(const uint8_t* __restrict__ linkX, int ix, int iy, int iz, int Nx, int Ny, int Nz) {
    return u8frac(linkX[idx(ix, iy, iz)]);
}

__device__ inline float link_ym(const uint8_t* __restrict__ linkY, int ix, int iy, int iz, int Nx, int Ny, int Nz, uint8_t PBC) {
    if (iy > 0) {
        return u8frac(linkY[idx(ix, iy - 1, iz)]);
    }
    return PBCy ? u8frac(linkY[idx(ix, Ny - 1, iz)]) : 0.0f;
}

__device__ inline float link_yp(const uint8_t* __restrict__ linkY, int ix, int iy, int iz, int Nx, int Ny, int Nz) {
    return u8frac(linkY[idx(ix, iy, iz)]);
}

__device__ inline float link_zm(const uint8_t* __restrict__ linkZ, int ix, int iy, int iz, int Nx, int Ny, int Nz, uint8_t PBC) {
    if (iz > 0) {
        return u8frac(linkZ[idx(ix, iy, iz - 1)]);
    }
    return PBCz ? u8frac(linkZ[idx(ix, iy, Nz - 1)]) : 0.0f;
}

__device__ inline float link_zp(const uint8_t* __restrict__ linkZ, int ix, int iy, int iz, int Nx, int Ny, int Nz) {
    return u8frac(linkZ[idx(ix, iy, iz)]);
}

extern "C" __global__ void
addexchange_cutcell(float* __restrict__ Bx, float* __restrict__ By, float* __restrict__ Bz,
                    float* __restrict__ mx, float* __restrict__ my, float* __restrict__ mz,
                    float* __restrict__ phi,
                    float* __restrict__ Ms_, float Ms_mul,
                    float* __restrict__ aLUT2d, uint8_t* __restrict__ regions,
                    uint8_t* __restrict__ linkX, uint8_t* __restrict__ linkY, uint8_t* __restrict__ linkZ,
                    float wx, float wy, float wz, float phiFloor, int Nx, int Ny, int Nz, uint8_t PBC) {

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

    float phi0 = phi[I];
    if (phi0 <= 0.0f) {
        return;
    }

    float denom = phi0;
    if (phiFloor > 0.0f) {
        denom = fmaxf(phi0, phiFloor);
    }

    uint8_t r0 = regions[I];
    float3 B = make_float3(0.0f, 0.0f, 0.0f);

    int i_;
    float3 m_;
    float a__;
    float link;

    link = link_xm(linkX, ix, iy, iz, Nx, Ny, Nz, PBC);
    if (link > 0.0f) {
        i_ = idx(lclampx(ix - 1), iy, iz);
        m_ = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(m_)) {
            a__ = aLUT2d[symidx(r0, regions[i_])];
            B += (link * wx * a__) * (m_ - m0);
        }
    }

    link = link_xp(linkX, ix, iy, iz, Nx, Ny, Nz);
    if (link > 0.0f) {
        i_ = idx(hclampx(ix + 1), iy, iz);
        m_ = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(m_)) {
            a__ = aLUT2d[symidx(r0, regions[i_])];
            B += (link * wx * a__) * (m_ - m0);
        }
    }

    link = link_ym(linkY, ix, iy, iz, Nx, Ny, Nz, PBC);
    if (link > 0.0f) {
        i_ = idx(ix, lclampy(iy - 1), iz);
        m_ = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(m_)) {
            a__ = aLUT2d[symidx(r0, regions[i_])];
            B += (link * wy * a__) * (m_ - m0);
        }
    }

    link = link_yp(linkY, ix, iy, iz, Nx, Ny, Nz);
    if (link > 0.0f) {
        i_ = idx(ix, hclampy(iy + 1), iz);
        m_ = make_float3(mx[i_], my[i_], mz[i_]);
        if (!is0(m_)) {
            a__ = aLUT2d[symidx(r0, regions[i_])];
            B += (link * wy * a__) * (m_ - m0);
        }
    }

    if (Nz != 1) {
        link = link_zm(linkZ, ix, iy, iz, Nx, Ny, Nz, PBC);
        if (link > 0.0f) {
            i_ = idx(ix, iy, lclampz(iz - 1));
            m_ = make_float3(mx[i_], my[i_], mz[i_]);
            if (!is0(m_)) {
                a__ = aLUT2d[symidx(r0, regions[i_])];
                B += (link * wz * a__) * (m_ - m0);
            }
        }

        link = link_zp(linkZ, ix, iy, iz, Nx, Ny, Nz);
        if (link > 0.0f) {
            i_ = idx(ix, iy, hclampz(iz + 1));
            m_ = make_float3(mx[i_], my[i_], mz[i_]);
            if (!is0(m_)) {
                a__ = aLUT2d[symidx(r0, regions[i_])];
                B += (link * wz * a__) * (m_ - m0);
            }
        }
    }

    float invMs = inv_Msat(Ms_, Ms_mul, I);
    Bx[I] += B.x * invMs / denom;
    By[I] += B.y * invMs / denom;
    Bz[I] += B.z * invMs / denom;
}
