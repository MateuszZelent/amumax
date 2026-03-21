package engine

// Averaging of quantities over entire universe or just magnet.

import (
	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
)

// average of quantity over universe
func qAverageUniverse(q Quantity) []float64 {
	if s, ok := q.(interface {
		Slice() (*data.Slice, bool)
	}); ok {
		slice, recycle := s.Slice()
		if recycle {
			defer cuda.Recycle(slice)
		}
		return sAverageUniverse(slice)
	}

	s := ValueOf(q)
	defer cuda.Recycle(s)
	return sAverageUniverse(s)
}

// average of slice over universe
func sAverageUniverse(s *data.Slice) []float64 {
	nCell := float64(prod(s.Size()))
	if !s.GPUAccess() {
		host := s.Host()
		avg := make([]float64, len(host))
		for c, values := range host {
			var sum float64
			for _, v := range values {
				sum += float64(v)
			}
			avg[c] = sum / nCell
			checkNaN1(avg[c])
		}
		return avg
	}

	sums := cuda.SumComponents(s)
	avg := make([]float64, len(sums))
	for i, sum := range sums {
		avg[i] = float64(sum) / nCell
		checkNaN1(avg[i])
	}
	return avg
}

// average of slice over the magnet volume
func sAverageMagnet(s *data.Slice) []float64 {
	if Geometry.Gpu().IsNil() {
		return sAverageUniverse(s)
	}
	avg := make([]float64, s.NComp())
	for i := range avg {
		avg[i] = float64(cuda.Dot(s.Comp(i), Geometry.Gpu())) / magnetNCell()
		checkNaN1(avg[i])
	}
	return avg
}

// number of cells in the magnet.
// not necessarily integer as cells can have fractional volume.
func magnetNCell() float64 {
	if Geometry.Gpu().IsNil() {
		return float64(GetMesh().NCell())
	}
	return float64(cuda.Sum(Geometry.Gpu()))
}
