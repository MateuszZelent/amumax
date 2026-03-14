package engine

import (
	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
)

var (
	extPhi   = newScalarField("ext_phi", "rad", "Azimuthal angle", setPhi)
	extTheta = newScalarField("ext_theta", "rad", "Polar angle", setTheta)
)

func setPhi(dst *data.Slice) {
	cuda.SetPhi(dst, NormMag.Buffer())
}

func setTheta(dst *data.Slice) {
	cuda.SetTheta(dst, NormMag.Buffer())
}
