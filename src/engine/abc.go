package engine

import (
	"math"
	"strings"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/log"
)

// SpongeAlpha is an additional per-cell damping that is added to Alpha.
// It is meant for Absorbing Boundary Conditions without consuming regions.
var SpongeAlpha = newScalarExcitation("SpongeAlpha", "",
	"Extra damping for Absorbing Boundary Conditions (added to Alpha)")

const mu0ABC = 4 * math.Pi * 1e-7

type abcSideSpec struct {
	XMinus bool
	XPlus  bool
	YMinus bool
	YPlus  bool
}

type abcSideWidths struct {
	XMinus float64
	XPlus  float64
	YMinus float64
	YPlus  float64
}

type abcDispersionParams struct {
	Mode     string
	Ms       float64
	Aex      float64
	D        float64
	Heff     float64
	DMI      float64
	SideSign float64
}

type abcDispersionState struct {
	Omega float64
	H1    float64
	H2    float64
}

type abcDesignResult struct {
	Valid        bool
	TotalWidth   float64
	RampWidth    float64
	PlateauWidth float64
	ThetaDeg     float64
	LambdaMinN   float64
	LambdaMaxN   float64
	KMinN        float64
	KMaxN        float64
	SGWorst      float64
	SAWorst      float64
	Heff         float64
	FMRGHz       float64
	Note         string
}

type abcHeffEstimate struct {
	BExtMag   float64
	HExt      float64
	HAnis     float64
	Heuristic float64
	Projected float64
	Used      float64
	MAvgNorm  float64
	Source    string
	Note      string
}

// spongeAlphaMSlice returns an MSlice for SpongeAlpha.
// If SpongeAlpha is zero, returns a nil-pointer MSlice (kernel treats NULL as 0).
func spongeAlphaMSlice() cuda.MSlice {
	if SpongeAlpha.isZero() {
		return cuda.MakeMSlice(data.NilSlice(1, GetMesh().Size()), []float64{0})
	}
	return SpongeAlpha.MSlice()
}

func init() {
	DeclFunc("ext_SetAbsorbingBoundary", SetAbsorbingBoundary,
		`Set absorbing boundary. Args: width(m), maxAlpha, direction, profile, param.
         Backward-compatible manual mode.
         direction supports legacy aliases "x", "y", "xy" and also side-specific tokens like
         "x-", "x+", "y-", "y+" or comma-separated combinations, e.g. "x-,x+".
         profile supports "smootherstep", "tanh", "power", "linear".
         param meaning depends on profile:
           "smootherstep": ignored
           "tanh":         steepness (higher = sharper transition, e.g. 4-10)
           "power":        exponent  (e.g. 2 = quadratic)
           "linear":       ignored
         Calling this again replaces the previous ABC (does not stack).`)

	DeclFunc("ext_SetAbsorbingBoundaryAdvanced", SetAbsorbingBoundaryAdvanced,
		`Advanced manual ABC with ramp + plateau.
         Args: totalWidth(m), rampWidth(m), maxAlpha, direction, profile, param.
         The outer part (totalWidth-rampWidth) is a plateau with alpha=maxAlpha.
         The inner part of width rampWidth is the smooth matching section.
         Use profile="smootherstep" or profile="power" with exponent 2 for best matching.`)

	DeclFunc("ext_ClearAbsorbingBoundary", ClearAbsorbingBoundary,
		"Remove all absorbing boundary conditions, resetting SpongeAlpha to zero.")

	DeclFunc("ext_AutoAbsorbingBoundary", AutoAbsorbingBoundary,
		`Legacy heuristic ABC from max frequency only.
         Uses a shortest-wavelength estimate and sets a smooth tanh profile.
         Args: maxFreqGHz, direction, maxAlpha, nWavelengths.
         This mode is quick, but it does NOT account for long-wavelength / small-k matching.`)

	DeclFunc("ext_AutoAbsorbingBoundaryAdvanced", AutoAbsorbingBoundaryAdvanced,
		`Physics-based automatic ABC design.
         Args:
           fMinGHz, fMaxGHz,
           direction,
           mode("BV","FV","DV","MSSW","DE"),
           maxAlpha,
           targetRLdB,
           targetEdgeAmpdB,
           thetaDeg,
           ku1Jm3,
           dmiJm2.

         Notes:
         - Reads Msat, Aex, Alpha and |B_ext| from region 0.
         - Estimates H_eff from the projected region-0 effective field when the
           current region-0 state is sufficiently aligned; otherwise falls back to
           H_ext + H_K.
         - The projected estimate is skipped when the region-0 temperature is non-zero
           and the engine is not relaxing, to avoid contamination by thermal field noise.
         - The projected H_eff estimate is intended for a relaxed or otherwise
           fairly uniform region-0 state.
         - ku1Jm3 is converted to an effective easy-axis field 2Ku1/(mu0*Ms).
         - dmiJm2 is an optional interfacial DMI correction used as a linear k-shift.
         - For single-axis designs, thetaDeg is the propagation angle with respect to
           the active boundary normal.
         - If both x and y boundaries are designed in one call, thetaDeg is interpreted
           as the global in-plane propagation angle measured from +x. The x sides use
           theta=thetaDeg and the y sides use theta=90-thetaDeg.
         - FV/DV currently uses an approximate forward-volume model and should be
           treated as experimental.
         - DV is treated as an alias of FV. DE is treated as an alias of MSSW.
         - The algorithm designs ramp width from the adiabatic condition and plateau width
           from the integrated attenuation target.`)

	DeclFunc("ext_AutoAbsorbingBoundaryAdvancedWithHeff", AutoAbsorbingBoundaryAdvancedWithHeff,
		`Physics-based automatic ABC design with explicit H_eff override.
         Args:
           fMinGHz, fMaxGHz,
           direction,
           mode("BV","FV","DV","MSSW","DE"),
           maxAlpha,
           targetRLdB,
           targetEdgeAmpdB,
           thetaDeg,
           heffApm,
           ku1Jm3,
           dmiJm2.

         Notes:
         - Uses the same algorithm as ext_AutoAbsorbingBoundaryAdvanced.
         - Bypasses the automatic H_eff estimate and uses heffApm directly.
         - heffApm may be any finite value, including zero or negative values.
         - Uses the same thetaDeg interpretation rules as ext_AutoAbsorbingBoundaryAdvanced.`)

	DeclFunc("ext_AutoAbsorbingBoundaryAdvancedFromRegion0", AutoAbsorbingBoundaryAdvancedFromRegion0,
		`Physics-based automatic ABC design using region-0 material parameters directly.
         Args:
           fMinGHz, fMaxGHz,
           direction,
           mode("BV","FV","DV","MSSW","DE"),
           maxAlpha,
           targetRLdB,
           targetEdgeAmpdB,
           thetaDeg.

         Notes:
         - Reads Msat, Aex, Alpha, |B_ext|, Ku1 and Dind from region 0.
         - Uses the same design algorithm as ext_AutoAbsorbingBoundaryAdvanced.
         - Uses the same thetaDeg interpretation rules as ext_AutoAbsorbingBoundaryAdvanced.
         - DV is treated as an alias of FV. DE is treated as an alias of MSSW.`)

	DeclFunc("ext_AutoAbsorbingBoundaryAdvancedFromRegion0WithHeff", AutoAbsorbingBoundaryAdvancedFromRegion0WithHeff,
		`Physics-based automatic ABC design using region-0 material parameters plus
         an explicit H_eff override.
         Args:
           fMinGHz, fMaxGHz,
           direction,
           mode("BV","FV","DV","MSSW","DE"),
           maxAlpha,
           targetRLdB,
           targetEdgeAmpdB,
           thetaDeg,
           heffApm.

         Notes:
         - Reads Msat, Aex, Alpha, |B_ext|, Ku1 and Dind from region 0.
         - Uses the same design algorithm as ext_AutoAbsorbingBoundaryAdvancedWithHeff.
         - heffApm may be any finite value, including zero or negative values.
         - Uses the same thetaDeg interpretation rules as ext_AutoAbsorbingBoundaryAdvanced.`)
}

// --- Profiles ---
// Each takes t ∈ [0,1] where:
//   t = 0 : bulk-side start of the ramp
//   t = 1 : outer edge of the simulation box
// and returns a normalized value in [0,1].

func profileLinear(t, _ float64) float64 {
	return clamp01(t)
}

func profilePower(t, power float64) float64 {
	t = clamp01(t)
	power = normalizedPower(power)
	return math.Pow(t, power)
}

// tanh profile: normalized S-curve centered at t = 0.5.
func profileTanh(t, steepness float64) float64 {
	t = clamp01(t)
	steepness = normalizedTanhSteepness(steepness)
	raw := math.Tanh(steepness * (t - 0.5))
	lo := math.Tanh(-0.5 * steepness)
	hi := math.Tanh(0.5 * steepness)
	return clamp01((raw - lo) / (hi - lo))
}

// smootherstep gives zero slope and zero curvature at both ends:
// p(t)=6t^5-15t^4+10t^3.
// This is usually a safer default for matching than linear or abrupt profiles.
func profileSmootherstep(t, _ float64) float64 {
	t = clamp01(t)
	return t * t * t * (t*(t*6-15) + 10)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func normalizedPower(power float64) float64 {
	if power < 1 {
		return 2
	}
	return power
}

func normalizedTanhSteepness(steepness float64) float64 {
	if steepness <= 0 {
		return 4
	}
	return steepness
}

func selectProfile(name string) func(t, param float64) float64 {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "smootherstep", "smooth", "smoothstep":
		return profileSmootherstep
	case "tanh":
		return profileTanh
	case "power", "pow", "parabolic", "quadratic":
		return profilePower
	case "linear", "lin":
		return profileLinear
	default:
		log.Log.ErrAndExit("Unknown ABC profile: %q. Use \"smootherstep\", \"tanh\", \"power\", or \"linear\".", name)
		return nil
	}
}

func averageProfileWeight(name string, param float64) float64 {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "power", "pow", "parabolic", "quadratic":
		return 1.0 / (normalizedPower(param) + 1.0)
	case "linear", "lin":
		return 0.5
	case "tanh":
		return 0.5
	case "smootherstep", "smooth", "smoothstep":
		return 0.5
	default:
		return 0.5
	}
}

func maxProfileSlope(name string, param float64) float64 {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "power", "pow", "parabolic", "quadratic":
		return normalizedPower(param)
	case "linear", "lin":
		return 1.0
	case "tanh":
		s := normalizedTanhSteepness(param)
		den := math.Tanh(0.5*s) - math.Tanh(-0.5*s)
		if den == 0 {
			return 1.0
		}
		return s / den
	case "smootherstep", "smooth", "smoothstep":
		return 1.875 // max of 30 t^2 (1-t)^2 at t=0.5
	default:
		return 1.0
	}
}

func appendABCNote(note, extra string) string {
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return strings.TrimSpace(note)
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return extra
	}
	if strings.Contains(note, extra) {
		return note
	}
	return note + "; " + extra
}

func parseBoundarySpec(direction string) abcSideSpec {
	raw := strings.ToLower(strings.TrimSpace(direction))
	raw = strings.ReplaceAll(raw, " ", "")
	raw = strings.ReplaceAll(raw, ";", ",")
	raw = strings.ReplaceAll(raw, "/", ",")

	spec := abcSideSpec{}
	switch raw {
	case "x":
		spec.XMinus, spec.XPlus = true, true
		return spec
	case "y":
		spec.YMinus, spec.YPlus = true, true
		return spec
	case "xy", "yx":
		spec.XMinus, spec.XPlus, spec.YMinus, spec.YPlus = true, true, true, true
		return spec
	}

	for _, tok := range strings.Split(raw, ",") {
		switch tok {
		case "x":
			spec.XMinus, spec.XPlus = true, true
		case "x-", "-x", "left":
			spec.XMinus = true
		case "x+", "+x", "right":
			spec.XPlus = true
		case "y":
			spec.YMinus, spec.YPlus = true, true
		case "y-", "-y", "bottom":
			spec.YMinus = true
		case "y+", "+y", "top":
			spec.YPlus = true
		case "xy", "yx":
			spec.XMinus, spec.XPlus, spec.YMinus, spec.YPlus = true, true, true, true
		}
	}

	// Very permissive fallback for mixed legacy strings.
	if !spec.any() {
		if strings.Contains(raw, "x") {
			spec.XMinus, spec.XPlus = true, true
		}
		if strings.Contains(raw, "y") {
			spec.YMinus, spec.YPlus = true, true
		}
	}

	return spec
}

func (s abcSideSpec) any() bool {
	return s.XMinus || s.XPlus || s.YMinus || s.YPlus
}

func roundUpToCells(width, cellSize float64, minCells int) float64 {
	if width <= 0 || cellSize <= 0 {
		return 0
	}
	cells := int(math.Ceil(width / cellSize))
	if cells < minCells {
		cells = minCells
	}
	return float64(cells) * cellSize
}

func normalizeSideWidths(total, ramp *abcSideWidths, spec abcSideSpec, worldX, worldY, cellX, cellY float64) {
	// Round to cell multiples and enforce minimal ramp discretization.
	total.XMinus = normalizeSingleWidth(total.XMinus, ramp.XMinus, cellX)
	total.XPlus = normalizeSingleWidth(total.XPlus, ramp.XPlus, cellX)
	total.YMinus = normalizeSingleWidth(total.YMinus, ramp.YMinus, cellY)
	total.YPlus = normalizeSingleWidth(total.YPlus, ramp.YPlus, cellY)
	ramp.XMinus = normalizeRampWidth(ramp.XMinus, cellX)
	ramp.XPlus = normalizeRampWidth(ramp.XPlus, cellX)
	ramp.YMinus = normalizeRampWidth(ramp.YMinus, cellY)
	ramp.YPlus = normalizeRampWidth(ramp.YPlus, cellY)

	if ramp.XMinus > total.XMinus {
		ramp.XMinus = total.XMinus
	}
	if ramp.XPlus > total.XPlus {
		ramp.XPlus = total.XPlus
	}
	if ramp.YMinus > total.YMinus {
		ramp.YMinus = total.YMinus
	}
	if ramp.YPlus > total.YPlus {
		ramp.YPlus = total.YPlus
	}

	clampAxisWidths(&total.XMinus, &total.XPlus, &ramp.XMinus, &ramp.XPlus, worldX, spec.XMinus && spec.XPlus)
	clampAxisWidths(&total.YMinus, &total.YPlus, &ramp.YMinus, &ramp.YPlus, worldY, spec.YMinus && spec.YPlus)
}

func normalizeSingleWidth(totalWidth, rampWidth, cellSize float64) float64 {
	if totalWidth <= 0 {
		return 0
	}
	minCells := 1
	if rampWidth > 0 {
		minCells = 2
	}
	return roundUpToCells(totalWidth, cellSize, minCells)
}

func normalizeRampWidth(rampWidth, cellSize float64) float64 {
	if rampWidth <= 0 {
		return 0
	}
	return roundUpToCells(rampWidth, cellSize, 2)
}

func clampAxisWidths(wNeg, wPos, rampNeg, rampPos *float64, worldSize float64, bothSides bool) {
	if worldSize <= 0 {
		*wNeg, *wPos = 0, 0
		*rampNeg, *rampPos = 0, 0
		return
	}

	maxSingle := 0.95 * worldSize
	if bothSides {
		maxSingle = 0.49 * worldSize
	}

	if *wNeg > maxSingle {
		*wNeg = maxSingle
	}
	if *wPos > maxSingle {
		*wPos = maxSingle
	}
	if *rampNeg > *wNeg {
		*rampNeg = *wNeg
	}
	if *rampPos > *wPos {
		*rampPos = *wPos
	}

	if bothSides {
		total := *wNeg + *wPos
		if total > 0.98*worldSize {
			scale := 0.98 * worldSize / total
			*wNeg *= scale
			*wPos *= scale
			*rampNeg = math.Min(*rampNeg, *wNeg)
			*rampPos = math.Min(*rampPos, *wPos)
		}
	}
}

// SetAbsorbingBoundary creates a smooth damping profile near boundaries.
// This is the backward-compatible wrapper: the whole width is treated as the ramp width.
func SetAbsorbingBoundary(width, maxAlpha float64, direction, profile string, param float64) {
	SetAbsorbingBoundaryAdvanced(width, width, maxAlpha, direction, profile, param)
}

// SetAbsorbingBoundaryAdvanced creates a smooth damping profile with an optional plateau.
// totalWidth = rampWidth + plateauWidth, where the plateau sits at the outer edge.
func SetAbsorbingBoundaryAdvanced(totalWidth, rampWidth, maxAlpha float64, direction, profile string, param float64) {
	spec := parseBoundarySpec(direction)
	if !spec.any() {
		log.Log.ErrAndExit("ext_SetAbsorbingBoundaryAdvanced: invalid direction %q", direction)
	}
	if totalWidth <= 0 {
		log.Log.ErrAndExit("ext_SetAbsorbingBoundaryAdvanced: totalWidth must be > 0")
	}
	if maxAlpha < 0 {
		log.Log.ErrAndExit("ext_SetAbsorbingBoundaryAdvanced: maxAlpha must be >= 0")
	}
	if rampWidth <= 0 || rampWidth > totalWidth {
		rampWidth = totalWidth
	}

	total := abcSideWidths{XMinus: totalWidth, XPlus: totalWidth, YMinus: totalWidth, YPlus: totalWidth}
	ramp := abcSideWidths{XMinus: rampWidth, XPlus: rampWidth, YMinus: rampWidth, YPlus: rampWidth}
	applyAbsorbingBoundary(spec, total, ramp, maxAlpha, profile, param)
}

func applyAbsorbingBoundary(spec abcSideSpec, total, ramp abcSideWidths, maxAlpha float64, profile string, param float64) {
	ClearAbsorbingBoundary()

	size := GetMesh().Size()
	cs := GetMesh().CellSize()
	ws := GetMesh().WorldSize()
	normalizeSideWidths(&total, &ramp, spec, ws[X], ws[Y], cs[X], cs[Y])
	prof := selectProfile(profile)

	maskHost := data.NewSlice(1, size)
	defer maskHost.Free()
	arr := maskHost.Scalars()

	for iz := 0; iz < size[Z]; iz++ {
		for iy := 0; iy < size[Y]; iy++ {
			for ix := 0; ix < size[X]; ix++ {
				xpos := (float64(ix) + 0.5) * cs[X]
				ypos := (float64(iy) + 0.5) * cs[Y]

				p := 0.0
				if spec.XMinus {
					p = combineNormalizedProfiles(p, normalizedBoundaryValue(xpos, total.XMinus, ramp.XMinus, prof, param))
				}
				if spec.XPlus {
					dist := ws[X] - xpos
					p = combineNormalizedProfiles(p, normalizedBoundaryValue(dist, total.XPlus, ramp.XPlus, prof, param))
				}
				if spec.YMinus {
					p = combineNormalizedProfiles(p, normalizedBoundaryValue(ypos, total.YMinus, ramp.YMinus, prof, param))
				}
				if spec.YPlus {
					dist := ws[Y] - ypos
					p = combineNormalizedProfiles(p, normalizedBoundaryValue(dist, total.YPlus, ramp.YPlus, prof, param))
				}

				arr[iz][iy][ix] = float32(maxAlpha * clamp01(p))
			}
		}
	}

	SpongeAlpha.AddGo(maskHost, nil)
}

func combineNormalizedProfiles(a, b float64) float64 {
	a = clamp01(a)
	b = clamp01(b)
	return 1 - (1-a)*(1-b)
}

// normalizedBoundaryValue returns a normalized damping fraction in [0,1].
// distToEdge is measured from the physical boundary inward.
func normalizedBoundaryValue(distToEdge, totalWidth, rampWidth float64,
	prof func(float64, float64) float64, param float64,
) float64 {
	if totalWidth <= 0 {
		return 0
	}
	if distToEdge < 0 {
		distToEdge = 0
	}
	if distToEdge >= totalWidth {
		return 0
	}
	if rampWidth <= 0 || rampWidth > totalWidth {
		rampWidth = totalWidth
	}

	plateauWidth := totalWidth - rampWidth
	if distToEdge <= plateauWidth {
		return 1
	}

	t := (totalWidth - distToEdge) / rampWidth
	return clamp01(prof(t, param))
}

// ClearAbsorbingBoundary removes all absorbing boundary conditions,
// resetting SpongeAlpha to zero everywhere.
func ClearAbsorbingBoundary() {
	SpongeAlpha.RemoveExtraTerms()
	SpongeAlpha.Set(0)
}

// AutoAbsorbingBoundary is kept as a legacy heuristic:
// it uses only the shortest wavelength at maxFreq and ignores the small-k / long-wave matching limit.
func AutoAbsorbingBoundary(maxFreqGHz float64, direction string, maxAlpha float64, nWavelengths float64) {
	ms := Msat.GetRegion(0)
	aex := Aex.GetRegion(0)
	alpha0 := Alpha.GetRegion(0)
	if ms == 0 {
		log.Log.ErrAndExit("ext_AutoAbsorbingBoundary: Msat is zero. Set material parameters first.")
	}
	if aex == 0 {
		log.Log.ErrAndExit("ext_AutoAbsorbingBoundary: Aex is zero. Set material parameters first.")
	}
	if nWavelengths <= 0 {
		nWavelengths = 3
	}

	bExt := BExt.perRegion.GetRegion(0)
	bMag := math.Sqrt(bExt[0]*bExt[0] + bExt[1]*bExt[1] + bExt[2]*bExt[2])
	hExt := bMag / mu0ABC
	d := float64(GetMesh().Size()[Z]) * GetMesh().CellSize()[Z]
	omegaMax := 2 * math.Pi * maxFreqGHz * 1e9
	cellX := GetMesh().CellSize()[X]
	cellY := GetMesh().CellSize()[Y]
	cellMin := math.Min(cellX, cellY)

	params := abcDispersionParams{
		Mode:     "bv",
		Ms:       ms,
		Aex:      aex,
		D:        d,
		Heff:     hExt,
		DMI:      0,
		SideSign: 1,
	}
	kMaxSearch := 0.8 * math.Pi / cellMin
	roots := findDispersionRoots(omegaMax, params, 1e3, kMaxSearch)
	if len(roots) == 0 {
		log.Log.ErrAndExit("ext_AutoAbsorbingBoundary: could not find a propagating root at %.3f GHz", maxFreqGHz)
	}
	kMax := roots[len(roots)-1]
	lambdaMin := 2 * math.Pi / kMax
	width := nWavelengths * lambdaMin
	width = roundUpToCells(width, cellMin, 10)

	log.Log.Info("┌─ ABC Legacy Auto-Configuration ───────────")
	log.Log.Info("│ Material: Msat = %.3e A/m, Aex = %.3e J/m, α = %.4f", ms, aex, alpha0)
	log.Log.Info("│ |B_ext|: %.4f T  → |H_ext| = %.3e A/m", bMag, hExt)
	log.Log.Info("│ Thickness: d = %.2f nm", d*1e9)
	log.Log.Info("│ Max frequency: %.2f GHz", maxFreqGHz)
	log.Log.Info("│ Heuristic mode: shortest λ only (legacy)")
	log.Log.Info("│ k_max = %.3e rad/m, λ_min = %.2f nm", kMax, lambdaMin*1e9)
	log.Log.Info("│ ABC width = %.2f nm (%.2f × λ_min)", width*1e9, nWavelengths)
	log.Log.Info("│ Profile = tanh(4), maxAlpha = %.3f, direction = %s", maxAlpha, direction)
	log.Log.Info("└────────────────────────────────────────────")

	SetAbsorbingBoundaryAdvanced(width, width, maxAlpha, direction, "tanh", 4)
}

// AutoAbsorbingBoundaryAdvanced designs the ramp width from the adiabatic matching condition
// and the plateau width from the integrated attenuation target.
//
// Design logic:
//  1. For the selected dispersion mode, scan f ∈ [fMin, fMax].
//  2. For every propagating root, compute:
//     P_A ≈ (H1+H2)/(2*sqrt(H1*H2)),
//     Sg  = ω P_A / (|v_g,n| k_n²),
//     Sa  = |v_g,n| / (ω P_A).
//  3. Worst-case small-k matching gives a characteristic ramp scale:
//     L_grad ~ C_RL * α_max * max|p'| * Sg*,
//     where C_RL = 1 + RL/10 is a conservative engineering factor.
//     Also enforce L_grad >= 2 λ_n,min.
//  4. Integrated attenuation gives the plateau width:
//     α_max ( <p> L_grad + L_plateau ) >= Sa* ln(10) * A_dB / 20.
func AutoAbsorbingBoundaryAdvanced(
	fMinGHz, fMaxGHz float64,
	direction, mode string,
	maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg, ku1Jm3, dmiJm2 float64,
) {
	autoAbsorbingBoundaryAdvanced(
		fMinGHz, fMaxGHz,
		direction, mode,
		maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg,
		ku1Jm3, dmiJm2,
		0, false,
		"ext_AutoAbsorbingBoundaryAdvanced",
	)
}

func AutoAbsorbingBoundaryAdvancedWithHeff(
	fMinGHz, fMaxGHz float64,
	direction, mode string,
	maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg, heffApm, ku1Jm3, dmiJm2 float64,
) {
	if math.IsNaN(heffApm) || math.IsInf(heffApm, 0) {
		log.Log.ErrAndExit("ext_AutoAbsorbingBoundaryAdvancedWithHeff: heffApm must be finite")
	}
	autoAbsorbingBoundaryAdvanced(
		fMinGHz, fMaxGHz,
		direction, mode,
		maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg,
		ku1Jm3, dmiJm2,
		heffApm, true,
		"ext_AutoAbsorbingBoundaryAdvancedWithHeff",
	)
}

func autoAbsorbingBoundaryAdvanced(
	fMinGHz, fMaxGHz float64,
	direction, mode string,
	maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg, ku1Jm3, dmiJm2, heffApm float64,
	useExplicitHeff bool,
	callerName string,
) {
	ms := Msat.GetRegion(0)
	aex := Aex.GetRegion(0)
	alpha0 := Alpha.GetRegion(0)
	if ms == 0 {
		log.Log.ErrAndExit("%s: Msat is zero. Set material parameters first.", callerName)
	}
	if aex == 0 {
		log.Log.ErrAndExit("%s: Aex is zero. Set material parameters first.", callerName)
	}
	if maxAlpha <= 0 {
		log.Log.ErrAndExit("%s: maxAlpha must be > 0", callerName)
	}
	if fMaxGHz <= 0 {
		log.Log.ErrAndExit("%s: fMaxGHz must be > 0", callerName)
	}
	if fMinGHz < 0 {
		fMinGHz = 0
	}
	if fMinGHz > fMaxGHz {
		fMinGHz, fMaxGHz = fMaxGHz, fMinGHz
	}
	if targetRLdB <= 0 {
		targetRLdB = 30
	}
	if targetEdgeAmpdB <= 0 {
		targetEdgeAmpdB = 40
	}

	spec := parseBoundarySpec(direction)
	if !spec.any() {
		log.Log.ErrAndExit("%s: invalid direction %q", callerName, direction)
	}

	mode = normalizeDispersionMode(mode)
	var heffEstimate abcHeffEstimate
	if useExplicitHeff {
		heffEstimate = explicitAutoABCHeffEstimate(ms, ku1Jm3, heffApm)
	} else {
		heffEstimate = estimateAutoABCHeff(ms, ku1Jm3)
	}
	if math.IsNaN(heffEstimate.Used) || math.IsInf(heffEstimate.Used, 0) {
		log.Log.ErrAndExit("%s: H_eff estimate is invalid (%.3e A/m). Relax the state first or use an explicit heffApm override.", callerName, heffEstimate.Used)
	}
	if heffEstimate.Used <= 0 {
		heffEstimate.Note = appendABCNote(heffEstimate.Note,
			"H_eff used is non-positive; proceeding because the simplified dispersion can still admit finite-k propagation, but low-k coverage may require manual validation")
	}
	d := float64(GetMesh().Size()[Z]) * GetMesh().CellSize()[Z]
	xThetaDeg, yThetaDeg, mixedAxisAngles := resolveAutoABCAxisAngles(spec, thetaDeg)

	var total abcSideWidths
	var ramp abcSideWidths
	var xMinusRes, xPlusRes, yMinusRes, yPlusRes abcDesignResult

	if spec.XMinus {
		res := designAutoABCSide(GetMesh().WorldSize()[X], GetMesh().CellSize()[X], d, ms, aex, heffEstimate.Used, fMinGHz, fMaxGHz, mode, maxAlpha, targetRLdB, targetEdgeAmpdB, xThetaDeg, dmiJm2, -1, spec.XMinus && spec.XPlus)
		if !res.Valid {
			log.Log.ErrAndExit("%s: x- design failed", callerName)
		}
		xMinusRes = res
		total.XMinus, ramp.XMinus = res.TotalWidth, res.RampWidth
	}
	if spec.XPlus {
		res := designAutoABCSide(GetMesh().WorldSize()[X], GetMesh().CellSize()[X], d, ms, aex, heffEstimate.Used, fMinGHz, fMaxGHz, mode, maxAlpha, targetRLdB, targetEdgeAmpdB, xThetaDeg, dmiJm2, +1, spec.XMinus && spec.XPlus)
		if !res.Valid {
			log.Log.ErrAndExit("%s: x+ design failed", callerName)
		}
		xPlusRes = res
		total.XPlus, ramp.XPlus = res.TotalWidth, res.RampWidth
	}
	if spec.YMinus {
		res := designAutoABCSide(GetMesh().WorldSize()[Y], GetMesh().CellSize()[Y], d, ms, aex, heffEstimate.Used, fMinGHz, fMaxGHz, mode, maxAlpha, targetRLdB, targetEdgeAmpdB, yThetaDeg, dmiJm2, -1, spec.YMinus && spec.YPlus)
		if !res.Valid {
			log.Log.ErrAndExit("%s: y- design failed", callerName)
		}
		yMinusRes = res
		total.YMinus, ramp.YMinus = res.TotalWidth, res.RampWidth
	}
	if spec.YPlus {
		res := designAutoABCSide(GetMesh().WorldSize()[Y], GetMesh().CellSize()[Y], d, ms, aex, heffEstimate.Used, fMinGHz, fMaxGHz, mode, maxAlpha, targetRLdB, targetEdgeAmpdB, yThetaDeg, dmiJm2, +1, spec.YMinus && spec.YPlus)
		if !res.Valid {
			log.Log.ErrAndExit("%s: y+ design failed", callerName)
		}
		yPlusRes = res
		total.YPlus, ramp.YPlus = res.TotalWidth, res.RampWidth
	}

	if mode == "fv" {
		log.Log.Warn("ABC advanced auto: FV/DV uses an approximate forward-volume model. Treat it as experimental and validate against RL or |Γ| benchmarks.")
	}
	log.Log.Info("┌─ ABC Physics-Based Auto-Configuration ────")
	logABCSideDesign("x-", xMinusRes)
	logABCSideDesign("x+", xPlusRes)
	logABCSideDesign("y-", yMinusRes)
	logABCSideDesign("y+", yPlusRes)
	log.Log.Info("│ Material: Msat = %.3e A/m, Aex = %.3e J/m, α = %.4f", ms, aex, alpha0)
	log.Log.Info("│ |B_ext| = %.4f T  → H_ext = %.3e A/m", heffEstimate.BExtMag, heffEstimate.HExt)
	log.Log.Info("│ Ku1 input = %.3e J/m^3 → H_K ≈ %.3e A/m", ku1Jm3, heffEstimate.HAnis)
	if isFinitePositive(heffEstimate.Projected) {
		log.Log.Info("│ Projected H_eff from region-0 state = %.3e A/m (|<m>_r0| = %.3f)", heffEstimate.Projected, heffEstimate.MAvgNorm)
	}
	log.Log.Info("│ H_eff used = %.3e A/m (%s)", heffEstimate.Used, heffEstimate.Source)
	log.Log.Info("│ Thickness = %.2f nm, mode = %s, DMI = %.3e J/m^2", d*1e9, strings.ToUpper(mode), dmiJm2)
	if mixedAxisAngles {
		log.Log.Info("│ thetaDeg input = %.2f deg interpreted as a global in-plane angle from +x", thetaDeg)
		log.Log.Info("│ Resolved wall-normal angles: thetaX = %.2f deg, thetaY = %.2f deg", xThetaDeg, yThetaDeg)
	} else {
		log.Log.Info("│ theta = %.2f deg relative to the active boundary normal", thetaDeg)
	}
	log.Log.Info("│ Band = [%.3f, %.3f] GHz, target RL = %.1f dB, target edge amplitude = %.1f dB", fMinGHz, fMaxGHz, targetRLdB, targetEdgeAmpdB)
	log.Log.Info("│ Profile = smootherstep, maxAlpha = %.3f", maxAlpha)
	if heffEstimate.Note != "" {
		log.Log.Warn("ABC advanced auto: %s", heffEstimate.Note)
	}
	log.Log.Info("└────────────────────────────────────────────")

	applyAbsorbingBoundary(spec, total, ramp, maxAlpha, "smootherstep", 0)
}

func AutoAbsorbingBoundaryAdvancedFromRegion0(
	fMinGHz, fMaxGHz float64,
	direction, mode string,
	maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg float64,
) {
	autoAbsorbingBoundaryAdvanced(
		fMinGHz, fMaxGHz,
		direction, mode,
		maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg,
		Ku1.GetRegion(0),
		Dind.GetRegion(0),
		0, false,
		"ext_AutoAbsorbingBoundaryAdvancedFromRegion0",
	)
}

func AutoAbsorbingBoundaryAdvancedFromRegion0WithHeff(
	fMinGHz, fMaxGHz float64,
	direction, mode string,
	maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg, heffApm float64,
) {
	if math.IsNaN(heffApm) || math.IsInf(heffApm, 0) {
		log.Log.ErrAndExit("ext_AutoAbsorbingBoundaryAdvancedFromRegion0WithHeff: heffApm must be finite")
	}
	autoAbsorbingBoundaryAdvanced(
		fMinGHz, fMaxGHz,
		direction, mode,
		maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg,
		Ku1.GetRegion(0),
		Dind.GetRegion(0),
		heffApm, true,
		"ext_AutoAbsorbingBoundaryAdvancedFromRegion0WithHeff",
	)
}

func logABCSideDesign(name string, res abcDesignResult) {
	if !res.Valid {
		return
	}
	log.Log.Info("│ %s: theta = %.2f deg, λn[min,max] = [%.2f, %.2f] nm, Lramp = %.2f nm, Lplateau = %.2f nm, Ltotal = %.2f nm",
		name, res.ThetaDeg, res.LambdaMinN*1e9, res.LambdaMaxN*1e9, res.RampWidth*1e9, res.PlateauWidth*1e9, res.TotalWidth*1e9)
	if res.Note != "" {
		log.Log.Info("│ %s: note: %s", name, res.Note)
	}
}

func baseAutoABCHeffEstimate(ms, ku1Jm3 float64) abcHeffEstimate {
	estimate := abcHeffEstimate{
		Source: "heuristic H_ext + H_K",
	}

	bExt := BExt.perRegion.GetRegion(0)
	estimate.BExtMag = math.Sqrt(bExt[0]*bExt[0] + bExt[1]*bExt[1] + bExt[2]*bExt[2])
	estimate.HExt = estimate.BExtMag / mu0ABC
	if ku1Jm3 != 0 {
		estimate.HAnis = 2 * ku1Jm3 / (mu0ABC * ms)
	}
	estimate.Heuristic = estimate.HExt + estimate.HAnis
	estimate.Used = estimate.Heuristic
	return estimate
}

func explicitAutoABCHeffEstimate(ms, ku1Jm3, heffApm float64) abcHeffEstimate {
	estimate := baseAutoABCHeffEstimate(ms, ku1Jm3)
	estimate.Used = heffApm
	estimate.Source = "explicit heffApm input"
	return estimate
}

func canUseProjectedAutoABCHeff() (bool, string) {
	if !relaxing {
		tempR0 := Temp.GetRegion(0)
		if math.Abs(tempR0) > 0 {
			return false, "projected H_eff was skipped because region-0 temperature is non-zero while not relaxing, so B_eff may include BTherm"
		}
	}
	return true, ""
}

func estimateAutoABCHeff(ms, ku1Jm3 float64) abcHeffEstimate {
	estimate := baseAutoABCHeffEstimate(ms, ku1Jm3)

	mAvg := NormMag.Region(0).Average()
	estimate.MAvgNorm = mAvg.Len()
	if estimate.MAvgNorm < 0.85 {
		estimate.Note = "|<m>_r0| < 0.85, so projected H_eff was skipped and H_ext + H_K was used"
		return estimate
	}

	allowed, why := canUseProjectedAutoABCHeff()
	if !allowed {
		estimate.Note = appendABCNote(estimate.Note, why)
		return estimate
	}

	mHat := mAvg.Div(estimate.MAvgNorm)
	bEffAvg := BEff.Region(0).Average()
	estimate.Projected = math.Abs(bEffAvg.Dot(mHat)) / mu0ABC
	if isFinitePositive(estimate.Projected) {
		estimate.Used = estimate.Projected
		estimate.Source = "projected region-0 <B_eff>·m"
		if estimate.MAvgNorm < 0.95 {
			estimate.Note = "region-0 |<m>| is below 0.95, so the projected H_eff estimate may be state-dependent"
		}
		return estimate
	}

	estimate.Note = "projected H_eff estimate was invalid, so H_ext + H_K was used"
	return estimate
}

func normalizeDispersionMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "bv", "backwardvolume", "backward-volume":
		return "bv"
	case "fv", "dv", "forwardvolume", "forward-volume", "volume":
		return "fv"
	case "mssw", "de", "damoneshbach", "damon-eshbach":
		return "mssw"
	default:
		log.Log.ErrAndExit("Unknown dispersion mode %q. Use BV, FV/DV, MSSW/DE.", mode)
		return ""
	}
}

func resolveAutoABCAxisAngles(spec abcSideSpec, thetaDeg float64) (xThetaDeg, yThetaDeg float64, mixedAxis bool) {
	xActive := spec.XMinus || spec.XPlus
	yActive := spec.YMinus || spec.YPlus
	if xActive && yActive {
		return thetaDeg, 90 - thetaDeg, true
	}
	return thetaDeg, thetaDeg, false
}

func designAutoABCSide(worldSize, cellSize, d, ms, aex, heff, fMinGHz, fMaxGHz float64, mode string,
	maxAlpha, targetRLdB, targetEdgeAmpdB, thetaDeg, dmiJm2, sideSign float64, sharedAxis bool,
) abcDesignResult {
	result := abcDesignResult{Heff: heff, ThetaDeg: thetaDeg}
	if worldSize <= 0 || cellSize <= 0 || d <= 0 || ms <= 0 || aex <= 0 {
		return result
	}

	const minNormalGroupVelocity = 1e-3
	absCos := math.Abs(math.Cos(thetaDeg * math.Pi / 180))
	note := ""
	if absCos < 0.05 {
		absCos = 0.05
		note = appendABCNote(note, "thetaDeg close to grazing incidence; |cos(theta)| clamped to 0.05")
	}

	params := abcDispersionParams{
		Mode:     mode,
		Ms:       ms,
		Aex:      aex,
		D:        d,
		Heff:     heff,
		DMI:      dmiJm2,
		SideSign: sideSign,
	}

	fmrOmega := abcDispersion(1e-12, params).Omega
	result.FMRGHz = fmrOmega / (2 * math.Pi * 1e9)

	kMinSearch := 1e2
	kMaxSearch := 0.8 * math.Pi / cellSize
	kDomainMinNormal := 2 * math.Pi / worldSize
	avgP := averageProfileWeight("smootherstep", 0)
	slopeMax := maxProfileSlope("smootherstep", 0)

	kMinNormal := math.Inf(+1)
	kMaxNormal := 0.0
	sgWorst := 0.0
	saWorst := 0.0
	used := 0

	const nFreq = 96
	for i := 0; i < nFreq; i++ {
		u := 0.0
		if nFreq > 1 {
			u = float64(i) / float64(nFreq-1)
		}
		frac := u * u
		fGHz := fMinGHz + frac*(fMaxGHz-fMinGHz)
		omegaTarget := 2 * math.Pi * fGHz * 1e9
		if omegaTarget <= 0 {
			continue
		}
		roots := findDispersionRoots(omegaTarget, params, kMinSearch, kMaxSearch)
		for _, k := range roots {
			st := abcDispersion(k, params)
			if st.Omega <= 0 {
				continue
			}
			vg := abcGroupVelocity(k, params)
			if math.IsNaN(vg) || math.IsInf(vg, 0) {
				continue
			}
			pa := abcEllipticityFactor(st.H1, st.H2)
			kn := math.Abs(k * absCos)
			if kn < kDomainMinNormal {
				kn = kDomainMinNormal
			}
			vgn := math.Abs(vg * absCos)
			if !isFinitePositive(vgn) {
				continue
			}
			if vgn < minNormalGroupVelocity {
				note = appendABCNote(note, "very small normal group velocity encountered; slow-mode branch may require manual validation")
				vgn = minNormalGroupVelocity
			}

			sg := (omegaTarget * pa) / (vgn * kn * kn)
			sa := vgn / (omegaTarget * pa)
			if sg > sgWorst {
				sgWorst = sg
			}
			if sa > saWorst {
				saWorst = sa
			}
			if kn < kMinNormal {
				kMinNormal = kn
			}
			if kn > kMaxNormal {
				kMaxNormal = kn
			}
			used++
		}
	}

	if used == 0 || !isFinitePositive(sgWorst) || !isFinitePositive(saWorst) {
		return result
	}

	lambdaMinN := 2 * math.Pi / kMaxNormal
	rlScale := 1.0 + targetRLdB/10.0
	lRamp := rlScale * maxAlpha * slopeMax * sgWorst
	if isFinitePositive(lambdaMinN) {
		lRamp = math.Max(lRamp, 2*lambdaMinN)
	}
	requiredAlphaIntegral := saWorst * targetEdgeAmpdB * math.Ln10 / 20.0
	lPlateau := requiredAlphaIntegral/maxAlpha - avgP*lRamp
	if lPlateau < 0 {
		lPlateau = 0
	}

	lRamp = roundUpToCells(lRamp, cellSize, 12)
	if lPlateau > 0 {
		lPlateau = roundUpToCells(lPlateau, cellSize, 6)
	}
	lTotal := lRamp + lPlateau

	maxAllowed := 0.95 * worldSize
	limitLabel := "95%"
	if sharedAxis {
		maxAllowed = 0.49 * worldSize
		limitLabel = "49%"
	}
	if lTotal > maxAllowed {
		limitNote := "total width clamped to " + limitLabel + " of the axis length"
		note = appendABCNote(note, limitNote)
		lTotal = maxAllowed
		if lRamp > lTotal {
			lRamp = lTotal
			lPlateau = 0
		} else {
			lPlateau = lTotal - lRamp
		}
	}

	result.Valid = true
	result.TotalWidth = lTotal
	result.RampWidth = lRamp
	result.PlateauWidth = lPlateau
	result.KMinN = kMinNormal
	result.KMaxN = kMaxNormal
	result.LambdaMaxN = 2 * math.Pi / kMinNormal
	result.LambdaMinN = lambdaMinN
	result.SGWorst = sgWorst
	result.SAWorst = saWorst
	result.Note = note
	return result
}

func isFinitePositive(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v > 0
}

func abcEllipticityFactor(h1, h2 float64) float64 {
	if h1 <= 0 || h2 <= 0 {
		return 1
	}
	return (h1 + h2) / (2 * math.Sqrt(h1*h2))
}

func abcDispersion(k float64, p abcDispersionParams) abcDispersionState {
	if k < 1e-12 {
		k = 1e-12
	}
	if p.Ms <= 0 {
		return abcDispersionState{}
	}
	hex := p.Heff + 2*p.Aex/(mu0ABC*p.Ms)*k*k
	kd := k * p.D
	F := dipoleFactor(kd)

	var h1, h2 float64
	switch p.Mode {
	case "bv":
		h1 = hex
		h2 = hex + p.Ms*F
	case "fv":
		h1 = hex
		h2 = hex + p.Ms*(1-F)
	case "mssw":
		expkd := math.Exp(-kd)
		h1 = hex + 0.5*p.Ms*(1-expkd)
		h2 = hex + 0.5*p.Ms*(1+expkd)
	default:
		h1 = hex
		h2 = hex + p.Ms*F
	}

	if h1 < 0 {
		h1 = 0
	}
	if h2 < 0 {
		h2 = 0
	}

	omega0 := gammaLL * mu0ABC * math.Sqrt(h1*h2)
	omega := omega0
	if p.DMI != 0 && p.Ms > 0 {
		omega += p.SideSign * 2 * gammaLL * p.DMI * k / p.Ms
	}
	if omega < 0 {
		omega = 0
	}
	return abcDispersionState{Omega: omega, H1: h1, H2: h2}
}

func dipoleFactor(kd float64) float64 {
	if kd < 1e-8 {
		return 1 - kd/2 + kd*kd/6
	}
	return (1 - math.Exp(-kd)) / kd
}

func abcGroupVelocity(k float64, p abcDispersionParams) float64 {
	dk := math.Max(1e-4*k, 1e3)
	k1 := k - dk
	if k1 < 1e-12 {
		k1 = 1e-12
	}
	k2 := k + dk
	w1 := abcDispersion(k1, p).Omega
	w2 := abcDispersion(k2, p).Omega
	return (w2 - w1) / (k2 - k1)
}

func findDispersionRoots(omegaTarget float64, p abcDispersionParams, kMin, kMax float64) []float64 {
	if omegaTarget <= 0 || kMin <= 0 || kMax <= kMin {
		return nil
	}
	const nSamples = 1024
	roots := make([]float64, 0, 4)
	logMin := math.Log(kMin)
	logMax := math.Log(kMax)

	prevK := kMin
	prevG := abcDispersion(prevK, p).Omega - omegaTarget
	for i := 1; i < nSamples; i++ {
		frac := float64(i) / float64(nSamples-1)
		k := math.Exp(logMin + frac*(logMax-logMin))
		g := abcDispersion(k, p).Omega - omegaTarget

		if math.IsNaN(g) || math.IsInf(g, 0) {
			prevK, prevG = k, g
			continue
		}

		if math.Abs(g) < 1e-6*omegaTarget {
			roots = appendRootUnique(roots, k)
		} else if !math.IsNaN(prevG) && !math.IsInf(prevG, 0) && prevG*g < 0 {
			root := bisectDispersionRoot(omegaTarget, p, prevK, k)
			roots = appendRootUnique(roots, root)
		}

		prevK, prevG = k, g
	}
	return roots
}

func bisectDispersionRoot(omegaTarget float64, p abcDispersionParams, kLo, kHi float64) float64 {
	gLo := abcDispersion(kLo, p).Omega - omegaTarget
	for i := 0; i < 80; i++ {
		kMid := 0.5 * (kLo + kHi)
		gMid := abcDispersion(kMid, p).Omega - omegaTarget
		if gLo*gMid <= 0 {
			kHi = kMid
		} else {
			kLo = kMid
			gLo = gMid
		}
	}
	return 0.5 * (kLo + kHi)
}

func appendRootUnique(roots []float64, root float64) []float64 {
	for _, r := range roots {
		if math.Abs(r-root) <= 1e-4*math.Max(root, 1) {
			return roots
		}
	}
	return append(roots, root)
}
