package engine

import (
	"math"
	"testing"
)

func TestGuideProjectionHostPreservesVolume(t *testing.T) {
	prevMesh := Mesh
	prevGuideProjectionEnabled := GuideProjectionEnabled
	prevGuideProjectionRefine := GuideProjectionRefine
	prevGuideProjectionHalo := GuideProjectionHalo
	prevGuideProjectionDS := GuideProjectionDS
	prevGuideProjectionDV := GuideProjectionDV
	prevGuideProjectionDW := GuideProjectionDW
	prevGuideProjectionUseCIC := GuideProjectionUseCIC
	prevGeomMode := GeomMode
	defer func() {
		Mesh = prevMesh
		GuideProjectionEnabled = prevGuideProjectionEnabled
		GuideProjectionRefine = prevGuideProjectionRefine
		GuideProjectionHalo = prevGuideProjectionHalo
		GuideProjectionDS = prevGuideProjectionDS
		GuideProjectionDV = prevGuideProjectionDV
		GuideProjectionDW = prevGuideProjectionDW
		GuideProjectionUseCIC = prevGuideProjectionUseCIC
		GeomMode = prevGeomMode
	}()

	Mesh = *GetMesh()
	Mesh.SetMesh(96, 16, 32, 2e-9, 2e-9, 2e-9, 0, 0, 0)
	Mesh.Create()

	GuideProjectionEnabled = true
	GuideProjectionRefine = 4
	GuideProjectionHalo = 2
	GuideProjectionDS = 0
	GuideProjectionDV = 0
	GuideProjectionDW = 0
	GuideProjectionUseCIC = true
	GeomMode = "cutcell"

	s := sinWaveguideNormal(120e-9, 12e-9, 8e-9, 60e-9, 8e-9, 0.1, 0)
	g := &geom{}
	result, ok := g.setGeomGuideProjectedHost(s)
	if !ok {
		t.Fatal("guide projection host path was not used")
	}
	if result.empty {
		t.Fatal("projected geometry is empty")
	}

	host := result.hostGeom.Host()[0]
	cell := Mesh.CellSize()
	cellVol := cell[X] * cell[Y] * cell[Z]
	var volume float64
	for _, phi := range host {
		volume += float64(phi) * cellVol
	}

	s0, s1 := s.guide.SRange()
	want := (s1 - s0) * s.guide.CrossSectionArea(0.5*(s0+s1))
	if math.Abs(volume-want)/want > 0.06 {
		t.Fatalf("projected volume mismatch: have=%g want=%g", volume, want)
	}
}
