package engine

import (
	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
)

var (
	extHopfIndexTwoPointStencil             = newScalarValue("ext_hopfindex_twopointstencil", "", "Hopf index calculated using two-point stencil", getHopfIndexTwoPointStencil)
	extHopfIndexDensityTwoPointStencil      = newScalarField("ext_hopfindexdensity_twopointstencil", "1/m3", "Hopf index density calculated using two-point stencil", setHopfIndexDensityTwoPointStencil)
	extEmergentMagneticFieldTwoPointStencil = newVectorField("ext_emergentmagneticfield_twopointstencil", "1/m2", "Emergent magnetic field calculated using two-point stencil", setEmergentMagneticFieldTwoPointStencil)

	extHopfIndexFivePointStencil             = newScalarValue("ext_hopfindex_fivepointstencil", "", "Hopf index calculated using five-point stencil", getHopfIndexFivePointStencil)
	extHopfIndexDensityFivePointStencil      = newScalarField("ext_hopfindexdensity_fivepointstencil", "1/m3", "Hopf index density calculated using five-point stencil", setHopfIndexDensityFivePointStencil)
	extEmergentMagneticFieldFivePointStencil = newVectorField("ext_emergentmagneticfield_fivepointstencil", "1/m2", "Emergent magnetic field calculated using five-point stencil", setEmergentMagneticFieldFivePointStencil)

	extHopfIndexSolidAngle             = newScalarValue("ext_hopfindex_solidangle", "", "Hopf index calculated using Berg-Lüscher lattice method", getHopfIndexSolidAngle)
	extHopfIndexDensitySolidAngle      = newScalarField("ext_hopfindexdensity_solidangle", "1/m3", "Hopf index density computed using Berg-Lüscher lattice method", setHopfIndexDensitySolidAngle)
	extEmergentMagneticFieldSolidAngle = newVectorField("ext_emergentmagneticfield_solidangle", "1/m2", "Emergent magnetic field computed using Berg-Lüscher lattice method", setEmergentMagneticFieldSolidAngle)

	extHopfIndexSolidAngleFourier = newScalarValue("ext_hopfindex_solidanglefourier", "", "Hopf index calculated using Berg-Lüscher lattice method to calculate emergent field, with emergent field Fourier transformed", getHopfIndexSolidAngleFourier)
)

func getHopfIndexTwoPointStencil() float64 {
	h := ValueOf(extHopfIndexDensityTwoPointStencil)
	defer cuda.Recycle(h)
	c := GetMesh().CellSize()
	return -c[X] * c[Y] * c[Z] * float64(cuda.Sum(h))
}

func setHopfIndexDensityTwoPointStencil(dst *data.Slice) {
	cuda.SetHopfIndexDensity_TwoPointStencil(dst, NormMag.Buffer(), NormMag.Mesh())
}

func setEmergentMagneticFieldTwoPointStencil(dst *data.Slice) {
	cuda.SetEmergentMagneticField_TwoPointStencil(dst, NormMag.Buffer(), NormMag.Mesh())
}

func getHopfIndexFivePointStencil() float64 {
	h := ValueOf(extHopfIndexDensityFivePointStencil)
	defer cuda.Recycle(h)
	c := GetMesh().CellSize()
	return -c[X] * c[Y] * c[Z] * float64(cuda.Sum(h))
}

func setHopfIndexDensityFivePointStencil(dst *data.Slice) {
	cuda.SetHopfIndexDensity_FivePointStencil(dst, NormMag.Buffer(), NormMag.Mesh())
}

func setEmergentMagneticFieldFivePointStencil(dst *data.Slice) {
	cuda.SetEmergentMagneticField_FivePointStencil(dst, NormMag.Buffer(), NormMag.Mesh())
}

func getHopfIndexSolidAngle() float64 {
	h := ValueOf(extHopfIndexDensitySolidAngle)
	defer cuda.Recycle(h)
	c := GetMesh().CellSize()
	return -c[X] * c[Y] * c[Z] * float64(cuda.Sum(h))
}

func setHopfIndexDensitySolidAngle(dst *data.Slice) {
	cuda.SetHopfIndexDensity_SolidAngle(dst, NormMag.Buffer(), NormMag.Mesh())
}

func setEmergentMagneticFieldSolidAngle(dst *data.Slice) {
	cuda.SetEmergentMagneticField_SolidAngle(dst, NormMag.Buffer(), NormMag.Mesh())
}

func getHopfIndexSolidAngleFourier() float64 {
	return cuda.GetHopfIndex_SolidAngleFourier(NormMag.Buffer(), NormMag.Mesh())
}
