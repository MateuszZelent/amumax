package engine

import (
	"math"
	"testing"
)

func TestSinGuideGeometryCenterlineProjection(t *testing.T) {
	g := newSinGuideGeometry(120e-9, 30e-9, 8e-9, 40e-9, 12e-9, 0.3, -5e-9)

	var prevS float64 = -1
	for _, x := range []float64{-45e-9, -10e-9, 15e-9, 42e-9} {
		z := g.centerZ(x)
		s, v, w, ok := g.ProjectPoint(x, 0, z)
		if !ok {
			t.Fatalf("ProjectPoint(%g, 0, %g) returned ok=false", x, z)
		}
		if math.Abs(v) > 1e-18 {
			t.Fatalf("expected v≈0 on centerline, got %g", v)
		}
		if math.Abs(w) > 1e-18 {
			t.Fatalf("expected w≈0 on centerline, got %g", w)
		}
		if s <= prevS {
			t.Fatalf("expected strictly increasing s, got prev=%g current=%g", prevS, s)
		}
		prevS = s

		frame := g.FrameAtS(s)
		if math.Abs(frame.R.X-x) > 1e-12 {
			t.Fatalf("frame x mismatch: want %g got %g", x, frame.R.X)
		}
		if math.Abs(frame.R.Z-z) > 1e-12 {
			t.Fatalf("frame z mismatch: want %g got %g", z, frame.R.Z)
		}
	}
}

func TestGuideTranslationPreservesLocalCoordinates(t *testing.T) {
	base := newArchGuideGeometry(100e-9, 24e-9, 6e-9, 18e-9, 3e-9)
	shifted := translatedGuideGeometry{base: base, dx: 7e-9, dy: -4e-9, dz: 11e-9}

	x := 18e-9
	y := 5e-9
	z := base.centerZ(x) + 1.2e-9

	s0, v0, w0, ok := base.ProjectPoint(x, y, z)
	if !ok {
		t.Fatal("base projection failed")
	}

	s1, v1, w1, ok := shifted.ProjectPoint(x+shifted.dx, y+shifted.dy, z+shifted.dz)
	if !ok {
		t.Fatal("translated projection failed")
	}

	if math.Abs(s0-s1) > 1e-15 {
		t.Fatalf("s mismatch after translation: want %g got %g", s0, s1)
	}
	if math.Abs(v0-v1) > 1e-15 {
		t.Fatalf("v mismatch after translation: want %g got %g", v0, v1)
	}
	if math.Abs(w0-w1) > 1e-15 {
		t.Fatalf("w mismatch after translation: want %g got %g", w0, w1)
	}

	frame0 := base.FrameAtS(s0)
	frame1 := shifted.FrameAtS(s1)
	if math.Abs((frame0.R.X+shifted.dx)-frame1.R.X) > 1e-15 {
		t.Fatalf("translated frame x mismatch: want %g got %g", frame0.R.X+shifted.dx, frame1.R.X)
	}
	if math.Abs((frame0.R.Y+shifted.dy)-frame1.R.Y) > 1e-15 {
		t.Fatalf("translated frame y mismatch: want %g got %g", frame0.R.Y+shifted.dy, frame1.R.Y)
	}
	if math.Abs((frame0.R.Z+shifted.dz)-frame1.R.Z) > 1e-15 {
		t.Fatalf("translated frame z mismatch: want %g got %g", frame0.R.Z+shifted.dz, frame1.R.Z)
	}
}
