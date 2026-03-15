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
		`Set absorbing boundary. Args: width(m), maxAlpha, direction("x","y","xy"), profile("tanh","linear","power"), param.
         param meaning depends on profile:
           "tanh":   steepness (higher = sharper transition, e.g. 4-10)
           "power":  exponent  (e.g. 2 = quadratic)
           "linear": ignored
         Calling this again replaces the previous ABC (does not stack).`)
	DeclFunc("ext_ClearAbsorbingBoundary", ClearAbsorbingBoundary,
		"Remove all absorbing boundary conditions, resetting SpongeAlpha to zero.")
	DeclFunc("ext_AutoAbsorbingBoundary", AutoAbsorbingBoundary,
		`Automatically configure ABC from material parameters and max frequency.
         Uses Kalinikos-Slavin dispersion to find shortest spin wave wavelength.
         Args: maxFreqGHz, direction("x","y","xy"), maxAlpha, nWavelengths.
         Example: ext_AutoAbsorbingBoundary(30, "x", 1.0, 3)`)
}

// --- Profile functions ---
// Each takes t ∈ [0,1] (0 = bulk edge of sponge, 1 = simulation boundary)
// and returns a value ∈ [0,1] that gets scaled by maxAlpha.

func profileLinear(t, _ float64) float64 { return t }

func profilePower(t, power float64) float64 { return math.Pow(t, power) }

// tanh profile: S-curve centered at t=0.5.
// steepness controls how sharp the transition is:
//   1-2: gentle S-curve (almost linear)
//   4-6: clear sigmoid shape
//   8+:  sharp step-like transition
func profileTanh(t, steepness float64) float64 {
	raw := math.Tanh(steepness * (t - 0.5))
	lo := math.Tanh(steepness * (-0.5))
	hi := math.Tanh(steepness * 0.5)
	return (raw - lo) / (hi - lo)
}

// SetAbsorbingBoundary creates a smooth damping profile near boundaries.
//
//	width:     thickness of the absorbing layer (m)
//	maxAlpha:  peak damping value at the simulation edge
//	direction: "x", "y", or "xy"
//	profile:   "linear", "power", or "tanh"
//	param:     profile-specific parameter (steepness for tanh, exponent for power, ignored for linear)
func SetAbsorbingBoundary(width, maxAlpha float64, direction, profile string, param float64) {
	// Clear any previous ABC so calling this multiple times replaces rather than stacks
	ClearAbsorbingBoundary()

	size := GetMesh().Size()
	cs := GetMesh().CellSize()
	ws := GetMesh().WorldSize()

	prof := selectProfile(profile)

	maskHost := data.NewSlice(1, size)
	defer maskHost.Free()
	arr := maskHost.Scalars()

	for iz := 0; iz < size[Z]; iz++ {
		for iy := 0; iy < size[Y]; iy++ {
			for ix := 0; ix < size[X]; ix++ {
				val := 0.0
				if strings.Contains(direction, "x") {
					xpos := (float64(ix) + 0.5) * cs[X]
					val = math.Max(val, boundaryValue(xpos, width, ws[X], maxAlpha, prof, param))
				}
				if strings.Contains(direction, "y") {
					ypos := (float64(iy) + 0.5) * cs[Y]
					val = math.Max(val, boundaryValue(ypos, width, ws[Y], maxAlpha, prof, param))
				}
				arr[iz][iy][ix] = float32(val)
			}
		}
	}
	SpongeAlpha.AddGo(maskHost, nil)
}

func selectProfile(name string) func(t, param float64) float64 {
	switch strings.ToLower(name) {
	case "tanh":
		return profileTanh
	case "power":
		return profilePower
	case "linear":
		return profileLinear
	default:
		log.Log.ErrAndExit("Unknown ABC profile: %q. Use \"tanh\", \"power\", or \"linear\".", name)
		return nil
	}
}

// boundaryValue returns the damping value for a cell at position pos
// along an axis of total length worldSize.
func boundaryValue(pos, width, worldSize, maxAlpha float64,
	prof func(float64, float64) float64, param float64) float64 {
	// Left boundary: pos ∈ [0, width]  →  t goes from 1 (edge) to 0 (bulk)
	if pos < width {
		t := 1.0 - pos/width
		return maxAlpha * prof(t, param)
	}
	// Right boundary: pos ∈ [worldSize-width, worldSize]
	if pos > worldSize-width {
		t := (pos - (worldSize - width)) / width
		return maxAlpha * prof(t, param)
	}
	return 0.0
}

// ClearAbsorbingBoundary removes all absorbing boundary conditions,
// resetting SpongeAlpha to zero everywhere.
func ClearAbsorbingBoundary() {
	SpongeAlpha.RemoveExtraTerms()
	SpongeAlpha.Set(0)
}

// AutoAbsorbingBoundary automatically configures ABC parameters using the
// Kalinikos-Slavin dispersion relation to determine the shortest spin wave
// wavelength at the given frequency.
//
//	maxFreqGHz:   maximum excitation frequency (GHz)
//	direction:    "x", "y", or "xy"
//	maxAlpha:     peak damping value at boundary edge
//	nWavelengths: number of shortest wavelengths for sponge width (typically 3-5)
func AutoAbsorbingBoundary(maxFreqGHz float64, direction string, maxAlpha float64, nWavelengths float64) {
	// Read material parameters from region 0
	ms := Msat.GetRegion(0) // A/m
	aex := Aex.GetRegion(0) // J/m
	alpha := Alpha.GetRegion(0)

	if ms == 0 {
		log.Log.ErrAndExit("ext_AutoAbsorbingBoundary: Msat is zero. Set material parameters first.")
	}
	if aex == 0 {
		log.Log.ErrAndExit("ext_AutoAbsorbingBoundary: Aex is zero. Set material parameters first.")
	}

	// External field: read B_ext from region 0 and compute |B|
	bExt := BExt.perRegion.GetRegion(0)
	bMag := math.Sqrt(bExt[0]*bExt[0] + bExt[1]*bExt[1] + bExt[2]*bExt[2])
	mu0 := 4 * math.Pi * 1e-7
	hExt := bMag / mu0 // external field in A/m

	// Film thickness
	d := float64(GetMesh().Size()[Z]) * GetMesh().CellSize()[Z]

	// Target angular frequency
	omegaMax := 2 * math.Pi * maxFreqGHz * 1e9

	// Exchange length factor: D_ex = 2*Aex / (μ₀*Ms)
	Dex := 2 * aex / (mu0 * ms)

	// Find k_max using bisection on KS dispersion (with external field)
	kMax := findKMax(omegaMax, ms, Dex, d, hExt)
	lambdaMin := 2 * math.Pi / kMax

	// Compute ABC width
	width := nWavelengths * lambdaMin

	// Ensure minimum width of 10 cells
	cellSize := GetMesh().CellSize()[X]
	if strings.Contains(direction, "y") && !strings.Contains(direction, "x") {
		cellSize = GetMesh().CellSize()[Y]
	}
	minWidth := 10 * cellSize
	if width < minWidth {
		log.Log.Warn("ABC width (%.1f nm) < 10 cells (%.1f nm), using minimum", width*1e9, minWidth*1e9)
		width = minWidth
	}

	// Check that width doesn't exceed half the domain
	for _, axis := range direction {
		var worldSize float64
		switch axis {
		case 'x':
			worldSize = GetMesh().WorldSize()[X]
		case 'y':
			worldSize = GetMesh().WorldSize()[Y]
		}
		if width > worldSize/2 {
			log.Log.Warn("ABC width (%.1f nm) exceeds half the %c domain (%.1f nm), clamping",
				width*1e9, axis, worldSize*1e9/2)
			width = worldSize / 2 * 0.95 // leave 5% gap
		}
	}

	// Log results
	log.Log.Info("┌─ ABC Auto-Configuration ──────────────────")
	log.Log.Info("│ Material:  Msat = %.3e A/m, Aex = %.3e J/m, α = %.4f", ms, aex, alpha)
	log.Log.Info("│ B_ext:     (%.4f, %.4f, %.4f) T  → |H| = %.3e A/m", bExt[0], bExt[1], bExt[2], hExt)
	log.Log.Info("│ Film thickness:  d = %.2f nm", d*1e9)
	log.Log.Info("│ Max frequency:   f = %.2f GHz  (ω = %.3e rad/s)", maxFreqGHz, omegaMax)
	log.Log.Info("│ KS dispersion:   k_max = %.3e rad/m", kMax)
	log.Log.Info("│ Shortest λ:      λ_min = %.2f nm", lambdaMin*1e9)
	log.Log.Info("│ ABC width:       %.2f nm  (%.1f × λ_min)", width*1e9, nWavelengths)
	log.Log.Info("│ Max damping:     α_sponge = %.2f", maxAlpha)
	log.Log.Info("│ Direction:       %s", direction)
	log.Log.Info("│ Profile:         tanh (steepness=4)")
	log.Log.Info("└────────────────────────────────────────────")

	SetAbsorbingBoundary(width, maxAlpha, direction, "tanh", 4)
}

// ksDispersionDE computes the angular frequency ω for a given wave vector k
// using the Kalinikos-Slavin dispersion relation for Damon-Eshbach (DE) mode.
//
// Formula: ω² = (γμ₀)² · (H + D·k²) · (H + D·k² + Ms·(1 - P(kd)))
// where:
//   H     = external field in A/m
//   D     = 2·Aex/(μ₀·Ms) exchange length factor
//   P(kd) = 1 - (1 - exp(-kd)) / (kd)
//
// For large k (exchange dominated), ω → γμ₀·D·k².
// For small k and H>0, ω → γμ₀·√(H·(H+Ms)) (FMR frequency).
func ksDispersionDE(k, ms, Dex, d, hExt float64) float64 {
	mu0 := 4 * math.Pi * 1e-7

	kd := k * d
	var P float64
	if kd < 1e-6 {
		P = kd / 2 // Taylor expansion for kd → 0
	} else {
		P = 1 - (1-math.Exp(-kd))/kd
	}

	Htot := hExt + Dex*k*k // internal field + exchange (A/m)

	// DE mode: ω² = (γμ₀)² · H_tot · (H_tot + Ms·(1-P))
	omegaSq := (gammaLL * mu0) * (gammaLL * mu0) * Htot * (Htot + ms*(1-P))
	if omegaSq < 0 {
		return 0
	}
	return math.Sqrt(omegaSq)
}

// findKMax finds the wave vector k at which ω(k) = omegaTarget
// using bisection on the KS dispersion relation.
func findKMax(omegaTarget, ms, Dex, d, hExt float64) float64 {
	// Upper bound: pure exchange limit k_max ≈ sqrt(ω/(γμ₀·D))
	mu0 := 4 * math.Pi * 1e-7
	kUpperEstimate := math.Sqrt(omegaTarget / (gammaLL * mu0 * Dex))
	kHi := kUpperEstimate * 2 // safety margin

	// Bisection
	kLo := 0.0
	for i := 0; i < 100; i++ {
		kMid := (kLo + kHi) / 2
		omega := ksDispersionDE(kMid, ms, Dex, d, hExt)
		if omega < omegaTarget {
			kLo = kMid
		} else {
			kHi = kMid
		}
	}
	return kHi
}
