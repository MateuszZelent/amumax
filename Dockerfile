# CUDA 12.4 for H100 (sm_90) support — cuRAND requires matching architecture kernels
FROM docker.io/nvidia/cuda:12.4.0-devel-ubuntu22.04
RUN apt-get update
RUN apt-get install -y wget git

# Installing go
ENV GO_VERSION=1.25.0
RUN wget https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz
RUN rm go$GO_VERSION.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:$PATH

WORKDIR /src

ENV GOPATH=/src/.go/path
ENV GOCACHE=/src/.go/cache
ENV CGO_CFLAGS="-I/usr/local/cuda/include/"  
ENV CGO_LDFLAGS="-lcufft -lcuda -lcurand -L/usr/local/cuda/lib64/stubs/ -Wl,-rpath -Wl,\$ORIGIN" 
ENV CGO_CFLAGS_ALLOW='(-fno-schedule-insns|-malign-double|-ffast-math)'
RUN ln -sf /usr/local/cuda/lib64/stubs/libcuda.so /usr/local/cuda/lib64/stubs/libcuda.so.1 && \
    ln -sf /usr/local/cuda/lib64/stubs/libcufft.so /usr/local/cuda/lib64/stubs/libcufft.so.1 && \
    ln -sf /usr/local/cuda/lib64/stubs/libcurand.so /usr/local/cuda/lib64/stubs/libcurand.so.1
ENV LD_LIBRARY_PATH=/usr/local/cuda/lib64/stubs:/usr/local/cuda/lib64:${LD_LIBRARY_PATH}

RUN git config --global --add safe.directory /src
CMD go build -v \
    -ldflags "-X github.com/MathieuMoalic/amumax/src/version.VERSION=$(date -u +'%Y.%m.%d')" \
    -o build/amumax
