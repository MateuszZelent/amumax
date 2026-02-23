// Oersted field: cross-product kernel multiplication in Fourier space.
//
// Computes B = J x K (cross-product convolution) in-place,
// where J and K are both already FFT'd.
//
// B_x = J_y * K_z - J_z * K_y
// B_y = J_z * K_x - J_x * K_z
// B_z = J_x * K_y - J_y * K_x
//
// All complex products are done manually on interleaved re/im floats.
// The result overwrites fftJx, fftJy, fftJz.
//
// Nx, Ny, Nz are the number of COMPLEX elements per dimension.

extern "C" __global__ void
oerstedkernmul3d(float* __restrict__  fftJx,  float* __restrict__  fftJy,  float* __restrict__  fftJz,
                 float* __restrict__  fftKx,  float* __restrict__  fftKy,  float* __restrict__  fftKz,
                 int Nx, int Ny, int Nz) {

    int ix = blockIdx.x * blockDim.x + threadIdx.x;
    int iy = blockIdx.y * blockDim.y + threadIdx.y;
    int iz = blockIdx.z * blockDim.z + threadIdx.z;

    if(ix >= Nx || iy >= Ny || iz >= Nz) {
        return;
    }

    int I = (iz*Ny + iy)*Nx + ix;
    int e = 2 * I;

    // Fetch complex J components
    float reJx = fftJx[e  ];
    float imJx = fftJx[e+1];
    float reJy = fftJy[e  ];
    float imJy = fftJy[e+1];
    float reJz = fftJz[e  ];
    float imJz = fftJz[e+1];

    // Fetch complex K components (pre-computed FFT of real-space kernel)
    float reKx = fftKx[e  ];
    float imKx = fftKx[e+1];
    float reKy = fftKy[e  ];
    float imKy = fftKy[e+1];
    float reKz = fftKz[e  ];
    float imKz = fftKz[e+1];

    // Complex multiplication helper: (a+ib)(c+id) = (ac-bd) + i(ad+bc)
    // Bx = Jy*Kz - Jz*Ky
    float reBx = (reJy*reKz - imJy*imKz) - (reJz*reKy - imJz*imKy);
    float imBx = (reJy*imKz + imJy*reKz) - (reJz*imKy + imJz*reKy);

    // By = Jz*Kx - Jx*Kz
    float reBy = (reJz*reKx - imJz*imKx) - (reJx*reKz - imJx*imKz);
    float imBy = (reJz*imKx + imJz*reKx) - (reJx*imKz + imJx*reKz);

    // Bz = Jx*Ky - Jy*Kx
    float reBz = (reJx*reKy - imJx*imKy) - (reJy*reKx - imJy*imKx);
    float imBz = (reJx*imKy + imJx*reKy) - (reJy*imKx + imJy*reKx);

    // Overwrite J with B
    fftJx[e  ] = reBx;
    fftJx[e+1] = imBx;
    fftJy[e  ] = reBy;
    fftJy[e+1] = imBy;
    fftJz[e  ] = reBz;
    fftJz[e+1] = imBz;
}
