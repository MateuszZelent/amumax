package engine

import "math"

type geomCellFlags uint8

const (
	geomCellEmpty geomCellFlags = 1 << iota
	geomCellFull
	geomCellBoundary
)

// geomCellMetrics stores conservative cut-cell geometry information on the solver grid.
// FaceFraction uses the same component order as Geometry.FaceBuffer:
// -X, +X, -Y, +Y, -Z, +Z.
type geomCellMetrics struct {
	VolumeFraction float32
	FaceFraction   [6]float32
	Flags          geomCellFlags
}

func newGeomCellMetrics(volume float32, face [6]float32) geomCellMetrics {
	metrics := geomCellMetrics{
		VolumeFraction: clampUnitFloat32(volume),
		FaceFraction:   face,
	}
	for i := range metrics.FaceFraction {
		metrics.FaceFraction[i] = clampUnitFloat32(metrics.FaceFraction[i])
	}
	metrics.Flags = classifyGeomCellMetrics(metrics)
	return metrics
}

func (m geomCellMetrics) Face(axis int, positive bool) float32 {
	index := axis * 2
	if positive {
		index++
	}
	return m.FaceFraction[index]
}

func classifyGeomCellMetrics(m geomCellMetrics) geomCellFlags {
	fullFaces := true
	for _, face := range m.FaceFraction {
		if !approxUnitFloat32(face, 1) {
			fullFaces = false
			break
		}
	}

	switch {
	case approxUnitFloat32(m.VolumeFraction, 0):
		return geomCellEmpty
	case approxUnitFloat32(m.VolumeFraction, 1) && fullFaces:
		return geomCellFull
	default:
		return geomCellBoundary
	}
}

func clampUnitFloat32(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func approxUnitFloat32(a, b float32) bool {
	return math.Abs(float64(a-b)) <= 1e-6
}
