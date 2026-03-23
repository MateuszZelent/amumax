package engine

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/fsutil"
	"github.com/MathieuMoalic/amumax/src/log"
)

var (
	GuideDiagBins      = 256
	GuideDiagSubsample = 2
	GuideDiagCutOnly   = true

	GuideSMap ScalarField
	GuideWMap ScalarField
)

func init() {
	GuideSMap = newScalarField("guide_s_map", "m", "Guide arc-length coordinate projected at each occupied cell center", setGuideSMap)
	GuideWMap = newScalarField("guide_w_map", "m", "Guide local thickness coordinate projected at each occupied cell center", setGuideWMap)
}

type guideProfile struct {
	S            []float64
	Vol          []float64
	Area         []float64
	AreaAnalytic []float64
	Zc           []float64
	Bex          []float64
	Bdem         []float64
	Beff         []float64
	Count        []float64
}

func setGuideSMap(dst *data.Slice) {
	setGuideCoordinateMap(dst, func(s, _, _ float64) float32 { return float32(s) })
}

func setGuideWMap(dst *data.Slice) {
	setGuideCoordinateMap(dst, func(_, _, w float64) float32 { return float32(w) })
}

func setGuideCoordinateMap(dst *data.Slice, project func(s, v, w float64) float32) {
	guide := Geometry.shape.guide
	hostDst := data.NewSlice(1, dst.Size())
	if guide == nil {
		data.Copy(dst, hostDst)
		return
	}

	geom, recycle := Geometry.Slice()
	if recycle {
		defer cuda.Recycle(geom)
	}
	hostGeom := geom.HostCopy()
	values := hostDst.Host()[0]
	geomValues := hostGeom.Host()[0]
	n := dst.Size()

	for iz := 0; iz < n[Z]; iz++ {
		for iy := 0; iy < n[Y]; iy++ {
			for ix := 0; ix < n[X]; ix++ {
				idx := data.Index(n, ix, iy, iz)
				if geomValues[idx] <= 0 {
					continue
				}
				r := index2Coord(ix, iy, iz)
				s, v, w, ok := guide.ProjectPoint(r[X], r[Y], r[Z])
				if !ok {
					continue
				}
				values[idx] = project(s, v, w)
			}
		}
	}

	data.Copy(dst, hostDst)
}

func saveGuideDiagnostics() {
	saveGuideDiagnosticsAs("guide")
}

func saveGuideDiagnosticsAs(prefix string) {
	profile, err := buildGuideProfile()
	if err != nil {
		log.Log.ErrAndExit("SaveGuideDiagnosticsAs: %v", err)
	}

	base := strings.TrimSpace(prefix)
	if base == "" {
		base = "guide"
	}

	write := func(suffix, header string, row func(i int) string) {
		filename := base + "_" + suffix + ".csv"
		if err := writeGuideProfileCSV(filename, header, len(profile.S), row); err != nil {
			log.Log.ErrAndExit("SaveGuideDiagnosticsAs: writing %s failed: %v", filename, err)
		}
	}

	write("area_s", "s_m,area_m2,analytic_area_m2,vol_m3,count\n", func(i int) string {
		return fmt.Sprintf("%.17g,%.17g,%.17g,%.17g,%.17g\n", profile.S[i], profile.Area[i], profile.AreaAnalytic[i], profile.Vol[i], profile.Count[i])
	})
	write("zc_s", "s_m,zc_m,vol_m3,count\n", func(i int) string {
		return fmt.Sprintf("%.17g,%.17g,%.17g,%.17g\n", profile.S[i], profile.Zc[i], profile.Vol[i], profile.Count[i])
	})
	write("bex_s", "s_m,bex_T,vol_m3,count\n", func(i int) string {
		return fmt.Sprintf("%.17g,%.17g,%.17g,%.17g\n", profile.S[i], profile.Bex[i], profile.Vol[i], profile.Count[i])
	})
	write("bdem_s", "s_m,bdem_T,vol_m3,count\n", func(i int) string {
		return fmt.Sprintf("%.17g,%.17g,%.17g,%.17g\n", profile.S[i], profile.Bdem[i], profile.Vol[i], profile.Count[i])
	})
	write("beff_s", "s_m,beff_T,vol_m3,count\n", func(i int) string {
		return fmt.Sprintf("%.17g,%.17g,%.17g,%.17g\n", profile.S[i], profile.Beff[i], profile.Vol[i], profile.Count[i])
	})
}

func buildGuideProfile() (*guideProfile, error) {
	guide := Geometry.shape.guide
	if guide == nil {
		return nil, fmt.Errorf("current geometry does not carry guide metadata")
	}

	bins := GuideDiagBins
	if bins <= 0 {
		bins = 256
	}
	subsample := GuideDiagSubsample
	if subsample <= 0 {
		subsample = 1
	}

	s0, s1 := guide.SRange()
	if !(s1 > s0) {
		return nil, fmt.Errorf("invalid guide arc-length range [%g, %g]", s0, s1)
	}

	profile := newGuideProfile(guide, bins, s0, s1)
	ds := (s1 - s0) / float64(bins)

	geom, recycleGeom := Geometry.Slice()
	if recycleGeom {
		defer cuda.Recycle(geom)
	}
	hostGeom := geom.HostCopy()
	geomValues := hostGeom.Host()[0]

	bex, err := hostVectorMagnitude(BExch)
	if err != nil {
		return nil, err
	}
	bdem, err := hostVectorMagnitude(BDemag)
	if err != nil {
		return nil, err
	}
	beff, err := hostVectorMagnitude(BEff)
	if err != nil {
		return nil, err
	}

	n := Geometry.Mesh().Size()
	cell := Geometry.Mesh().CellSize()
	cellVol := cell[0] * cell[1] * cell[2]
	fullTol := float32(math.Max(GeomTol, 1e-3))
	subCount := float64(subsample * subsample * subsample)
	subVol := cellVol / subCount

	for iz := 0; iz < n[Z]; iz++ {
		for iy := 0; iy < n[Y]; iy++ {
			for ix := 0; ix < n[X]; ix++ {
				idx := data.Index(n, ix, iy, iz)
				phi := geomValues[idx]
				if phi <= 0 {
					continue
				}

				bounds := boundsFromIndex(ix, iy, iz)
				bexMag := bex[idx]
				bdemMag := bdem[idx]
				beffMag := beff[idx]

				if GuideDiagCutOnly && phi >= 1-fullTol {
					x, y, z := bounds.midpoint()
					if Geometry.shape.contains(x, y, z) {
						profile.addSample(guide, s0, ds, x, y, z, cellVol*float64(phi), bexMag, bdemMag, beffMag)
						continue
					}
				}

				for sx := 0; sx < subsample; sx++ {
					tx := (float64(sx) + 0.5) / float64(subsample)
					for sy := 0; sy < subsample; sy++ {
						ty := (float64(sy) + 0.5) / float64(subsample)
						for sz := 0; sz < subsample; sz++ {
							tz := (float64(sz) + 0.5) / float64(subsample)
							x, y, z := bounds.samplePoint(tx, ty, tz)
							if !Geometry.shape.contains(x, y, z) {
								continue
							}
							profile.addSample(guide, s0, ds, x, y, z, subVol, bexMag, bdemMag, beffMag)
						}
					}
				}
			}
		}
	}

	profile.finalize()
	return profile, nil
}

func newGuideProfile(guide guideGeometry, bins int, s0, s1 float64) *guideProfile {
	profile := &guideProfile{
		S:            make([]float64, bins),
		Vol:          make([]float64, bins),
		Area:         make([]float64, bins),
		AreaAnalytic: make([]float64, bins),
		Zc:           make([]float64, bins),
		Bex:          make([]float64, bins),
		Bdem:         make([]float64, bins),
		Beff:         make([]float64, bins),
		Count:        make([]float64, bins),
	}
	ds := (s1 - s0) / float64(bins)
	for i := range profile.S {
		s := s0 + (float64(i)+0.5)*ds
		profile.S[i] = s
		profile.AreaAnalytic[i] = guide.CrossSectionArea(s)
	}
	return profile
}

func (p *guideProfile) addSample(guide guideGeometry, s0, ds, x, y, z, dV, bex, bdem, beff float64) {
	s, _, _, ok := guide.ProjectPoint(x, y, z)
	if !ok {
		return
	}
	bin := int(math.Floor((s - s0) / ds))
	if bin < 0 {
		bin = 0
	}
	if bin >= len(p.S) {
		bin = len(p.S) - 1
	}
	p.Vol[bin] += dV
	p.Area[bin] += dV / ds
	p.Zc[bin] += dV * z
	p.Bex[bin] += dV * bex
	p.Bdem[bin] += dV * bdem
	p.Beff[bin] += dV * beff
	p.Count[bin]++
}

func (p *guideProfile) finalize() {
	for i := range p.S {
		if p.Vol[i] == 0 {
			continue
		}
		invVol := 1 / p.Vol[i]
		p.Zc[i] *= invVol
		p.Bex[i] *= invVol
		p.Bdem[i] *= invVol
		p.Beff[i] *= invVol
	}
}

func hostVectorMagnitude(q Quantity) ([]float64, error) {
	buf := ValueOf(q)
	defer cuda.Recycle(buf)
	host := buf.HostCopy()
	values := host.Host()
	if len(values) != 3 {
		return nil, fmt.Errorf("%s is not a vector quantity", nameOf(q))
	}

	out := make([]float64, len(values[0]))
	for i := range out {
		bx := float64(values[0][i])
		by := float64(values[1][i])
		bz := float64(values[2][i])
		out[i] = math.Sqrt(bx*bx + by*by + bz*bz)
	}
	return out, nil
}

func writeGuideProfileCSV(filename, header string, n int, row func(i int) string) error {
	var builder strings.Builder
	builder.WriteString(header)
	for i := 0; i < n; i++ {
		builder.WriteString(row(i))
	}

	if !filepath.IsAbs(filename) {
		filename = OD() + filename
	}
	return fsutil.Put(filename, []byte(builder.String()))
}
