package cuda

import (
	"fmt"
	"unsafe"

	"github.com/MathieuMoalic/amumax/src/cuda/cu"
	"github.com/MathieuMoalic/amumax/src/log"
)

// Int32s is a flat GPU int32 buffer.
type Int32s struct {
	Ptr unsafe.Pointer
	Len int
}

func NewInt32s(length int) *Int32s {
	ptr := cu.MemAlloc(int64(length) * 4)
	cu.MemsetD32(cu.DevicePtr(ptr), 0, int64(length))
	return &Int32s{Ptr: unsafe.Pointer(uintptr(ptr)), Len: length}
}

func (b *Int32s) Upload(src []int32) {
	log.AssertMsg(b.Len == len(src), "Upload: Length mismatch between destination (gpu) and source (host) int32 data")
	if len(src) == 0 {
		return
	}
	MemCpyHtoD(b.Ptr, unsafe.Pointer(&src[0]), int64(b.Len)*4)
}

func (b *Int32s) Download(dst []int32) {
	log.AssertMsg(b.Len == len(dst), "Download: Length mismatch between source (gpu) and destination (host) int32 data")
	if len(dst) == 0 {
		return
	}
	MemCpyDtoH(unsafe.Pointer(&dst[0]), b.Ptr, int64(b.Len)*4)
}

func (b *Int32s) Set(index int, value int32) {
	if index < 0 || index >= b.Len {
		log.Log.PanicIfError(fmt.Errorf("Int32s.Set: index out of range: %d", index))
	}
	src := value
	MemCpyHtoD(unsafe.Pointer(uintptr(b.Ptr)+uintptr(index)*4), unsafe.Pointer(&src), 4)
}

func (b *Int32s) Get(index int) int32 {
	if index < 0 || index >= b.Len {
		log.Log.PanicIfError(fmt.Errorf("Int32s.Get: index out of range: %d", index))
	}
	var dst int32
	MemCpyDtoH(unsafe.Pointer(&dst), unsafe.Pointer(uintptr(b.Ptr)+uintptr(index)*4), 4)
	return dst
}

func (b *Int32s) Free() {
	if b.Ptr != nil {
		cu.MemFree(cu.DevicePtr(uintptr(b.Ptr)))
	}
	b.Ptr = nil
	b.Len = 0
}
