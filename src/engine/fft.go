package engine

import (
	"math"
	"sync"

	"github.com/MathieuMoalic/amumax/src/log"
)

// FftEnabled is set from the CLI --fft flag. When false, no FFT work is done.
var FftEnabled bool

// fftTracker holds the NUDFT accumulator state for real-time FFT computation.
type fftTracker struct {
	mu sync.Mutex

	quantities []Quantity // tracked quantities
	labels     []string  // component labels (e.g., "mx", "my", "mz")
	nComp      int       // total components across all quantities

	minFreq float64 // Hz
	maxFreq float64 // Hz
	dFreq   float64 // Hz
	nFreqs  int     // number of frequency bins

	// Precomputed frequency array (Hz) — avoids recomputation per step
	freqs []float64

	// Global accumulator (full-run spectrum)
	accumReal [][]float64 // [component][freqBin]
	accumImag [][]float64 // [component][freqBin]
	startTime float64     // simulation time when accumulation started

	// Segment accumulator (for spectrogram STFT windows)
	segReal     [][]float64 // [component][freqBin] current segment
	segImag     [][]float64 // [component][freqBin] current segment
	segStart    float64     // start time of current segment
	segDuration float64     // duration of each segment (seconds)

	// Spectrogram: history of segment spectra
	spectrogramHistory [][]float64 // [timeSlice][freqBin] for one component
	spectrogramTimes   []float64   // center-time of each segment
	spectrogramComp    int         // which component index for spectrogram
	spectrogramMaxLen  int         // max history length

	// Subsampling: don't compute FFT every step, accumulate signal history
	// and batch-process periodically
	stepCounter   int     // counts steps since last FFT batch
	stepsPerBatch int     // how many steps to accumulate before NUDFT batch
	signalBuf     [][]signalSample // [component][] buffered (t, signal, dt) samples

	// Baseline subtraction: subtract static m(t0) to isolate dynamics
	baseline     []float64 // [component] captured at first step
	baselineSet  bool      // whether baseline has been captured

	initialized bool
}

// signalSample holds one time-domain sample
type signalSample struct {
	t      float64
	signal float64
	dt     float64
}

// globalFft is the singleton FFT tracker instance.
var globalFft = &fftTracker{
	spectrogramMaxLen: 200,
	spectrogramComp:   1, // default to my (y-component)
	stepsPerBatch:     10, // process NUDFT every N steps
}

func init() {
	DeclFunc("FftTrack", FftTrack,
		`Track a quantity for real-time FFT. Args: quantity, minFreqGHz, maxFreqGHz, dFreqGHz.
         Example: FftTrack(m, 0, 30, 0.1)`)
}

// FftTrack registers a quantity for NUDFT tracking.
// freqs in GHz for user convenience.
func FftTrack(q Quantity, minFreqGHz, maxFreqGHz, dFreqGHz float64) {
	if !FftEnabled {
		log.Log.Warn("FftTrack called but --fft flag is not set. FFT is disabled.")
		return
	}

	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()

	globalFft.quantities = append(globalFft.quantities, q)
	nComp := q.NComp()

	// Generate labels
	name := nameOf(q)
	if nComp == 1 {
		globalFft.labels = append(globalFft.labels, name)
	} else {
		compNames := []string{"x", "y", "z"}
		for c := 0; c < nComp && c < 3; c++ {
			globalFft.labels = append(globalFft.labels, name+compNames[c])
		}
	}

	globalFft.minFreq = minFreqGHz * 1e9
	globalFft.maxFreq = maxFreqGHz * 1e9
	globalFft.dFreq = dFreqGHz * 1e9

	if globalFft.dFreq <= 0 {
		log.Log.ErrAndExit("FftTrack: dFreqGHz must be > 0")
	}

	globalFft.nFreqs = int((globalFft.maxFreq-globalFft.minFreq)/globalFft.dFreq) + 1
	globalFft.nComp = len(globalFft.labels)

	// Precompute frequency array
	globalFft.freqs = make([]float64, globalFft.nFreqs)
	for fi := 0; fi < globalFft.nFreqs; fi++ {
		globalFft.freqs[fi] = globalFft.minFreq + float64(fi)*globalFft.dFreq
	}

	// Allocate global accumulators
	globalFft.accumReal = make([][]float64, globalFft.nComp)
	globalFft.accumImag = make([][]float64, globalFft.nComp)
	// Allocate segment accumulators
	globalFft.segReal = make([][]float64, globalFft.nComp)
	globalFft.segImag = make([][]float64, globalFft.nComp)
	// Allocate signal buffers
	globalFft.signalBuf = make([][]signalSample, globalFft.nComp)
	globalFft.baseline = make([]float64, globalFft.nComp)
	globalFft.baselineSet = false
	for c := 0; c < globalFft.nComp; c++ {
		globalFft.accumReal[c] = make([]float64, globalFft.nFreqs)
		globalFft.accumImag[c] = make([]float64, globalFft.nFreqs)
		globalFft.segReal[c] = make([]float64, globalFft.nFreqs)
		globalFft.segImag[c] = make([]float64, globalFft.nFreqs)
		globalFft.signalBuf[c] = make([]signalSample, 0, globalFft.stepsPerBatch)
	}

	// Segment duration for spectrogram STFT windows.
	// Shorter = more time slices but worse frequency resolution per segment.
	// 0.2/dFreq gives dense updates (e.g., 2 ns for 0.1 GHz resolution).
	globalFft.segDuration = 0.2 / globalFft.dFreq
	globalFft.segStart = Time
	globalFft.startTime = Time
	globalFft.stepCounter = 0
	globalFft.spectrogramHistory = nil
	globalFft.spectrogramTimes = nil

	globalFft.initialized = true

	log.Log.Info("┌─ FFT tracking configured ─────────────────")
	log.Log.Info("│ Quantity:    %s (%d components)", name, nComp)
	log.Log.Info("│ Freq range:  %.2f – %.2f GHz", minFreqGHz, maxFreqGHz)
	log.Log.Info("│ Resolution:  %.3f GHz (%d bins)", dFreqGHz, globalFft.nFreqs)
	log.Log.Info("│ Batch size:  %d steps", globalFft.stepsPerBatch)
	log.Log.Info("│ Spectrogram segment: %.2f ns", globalFft.segDuration*1e9)
	log.Log.Info("└────────────────────────────────────────────")
}

// doFftStep is called from step() in run.go. No-op when FFT is disabled.
//
// Performance strategy:
//   1. Every step: collect spatially-averaged signal (cheap — GPU already
//      computed it, we just read 3 floats). Buffer the (t, signal, dt) tuple.
//   2. Every N steps: batch-process all buffered samples through the NUDFT
//      inner loop. This amortizes the sin/cos cost over N steps.
func doFftStep() {
	if !FftEnabled || !globalFft.initialized {
		return
	}

	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()

	t := Time
	dt := DtSi

	// Step 1: Buffer the signal and capture baseline on first step
	compIdx := 0
	for _, q := range globalFft.quantities {
		avg := qAverageUniverse(q)
		for c := 0; c < q.NComp(); c++ {
			// Capture baseline (static configuration) on first step
			if !globalFft.baselineSet {
				globalFft.baseline[compIdx] = avg[c]
			}

			globalFft.signalBuf[compIdx] = append(globalFft.signalBuf[compIdx], signalSample{
				t:      t,
				signal: avg[c],
				dt:     dt,
			})
			compIdx++
		}
	}
	if !globalFft.baselineSet {
		globalFft.baselineSet = true
	}
	globalFft.stepCounter++

	// Step 2: Batch-process when we have enough samples
	if globalFft.stepCounter >= globalFft.stepsPerBatch {
		processFftBatch()
		globalFft.stepCounter = 0
	}
}

// processFftBatch processes all buffered signal samples through the NUDFT.
// Called with mutex already held.
func processFftBatch() {
	nf := globalFft.nFreqs
	freqs := globalFft.freqs
	twoPi := 2 * math.Pi

	for c := 0; c < globalFft.nComp; c++ {
		samples := globalFft.signalBuf[c]
		if len(samples) == 0 {
			continue
		}

		accumR := globalFft.accumReal[c]
		accumI := globalFft.accumImag[c]
		segR := globalFft.segReal[c]
		segI := globalFft.segImag[c]

		// Baseline value for this component (static m(t0))
		bl := globalFft.baseline[c]

		for _, s := range samples {
			// Subtract baseline (static configuration) from signal
			sigDt := (s.signal - bl) * s.dt
			// Use math.Sincos for each frequency bin (2x faster than separate Sin+Cos)
			for fi := 0; fi < nf; fi++ {
				phase := -twoPi * freqs[fi] * s.t
				sinP, cosP := math.Sincos(phase)
				re := sigDt * cosP
				im := sigDt * sinP
				accumR[fi] += re
				accumI[fi] += im
				segR[fi] += re
				segI[fi] += im
			}
		}

		// Clear buffer
		globalFft.signalBuf[c] = globalFft.signalBuf[c][:0]
	}

	// Check if segment is complete → snapshot for spectrogram
	t := Time
	if t-globalFft.segStart >= globalFft.segDuration {
		c := globalFft.spectrogramComp
		if c < globalFft.nComp {
			segT := t - globalFft.segStart
			if segT <= 0 {
				segT = 1
			}
			mag := make([]float64, nf)
			for fi := 0; fi < nf; fi++ {
				re := globalFft.segReal[c][fi]
				im := globalFft.segImag[c][fi]
				mag[fi] = math.Sqrt(re*re+im*im) / segT
			}
			globalFft.spectrogramHistory = append(globalFft.spectrogramHistory, mag)
			globalFft.spectrogramTimes = append(globalFft.spectrogramTimes, globalFft.segStart+segT/2)

			// Trim to max length
			if len(globalFft.spectrogramHistory) > globalFft.spectrogramMaxLen {
				globalFft.spectrogramHistory = globalFft.spectrogramHistory[1:]
				globalFft.spectrogramTimes = globalFft.spectrogramTimes[1:]
			}
		}

		// Reset segment accumulators
		for ci := 0; ci < globalFft.nComp; ci++ {
			for fi := range globalFft.segReal[ci] {
				globalFft.segReal[ci][fi] = 0
				globalFft.segImag[ci][fi] = 0
			}
		}
		globalFft.segStart = t
	}
}

// GetFftSpectrum returns the normalized FFT magnitude spectrum for all components.
func GetFftSpectrum() [][]float64 {
	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()

	if !globalFft.initialized {
		return nil
	}

	// Flush any pending samples before returning
	if globalFft.stepCounter > 0 {
		processFftBatch()
		globalFft.stepCounter = 0
	}

	totalT := Time - globalFft.startTime
	if totalT <= 0 {
		totalT = 1
	}

	result := make([][]float64, globalFft.nComp)
	for c := 0; c < globalFft.nComp; c++ {
		mag := make([]float64, globalFft.nFreqs)
		for fi := 0; fi < globalFft.nFreqs; fi++ {
			re := globalFft.accumReal[c][fi]
			im := globalFft.accumImag[c][fi]
			mag[fi] = math.Sqrt(re*re+im*im) / totalT
		}
		result[c] = mag
	}
	return result
}

// GetFftFreqAxis returns the frequency axis in GHz.
func GetFftFreqAxis() []float64 {
	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()

	if !globalFft.initialized {
		return nil
	}

	axis := make([]float64, globalFft.nFreqs)
	for fi := 0; fi < globalFft.nFreqs; fi++ {
		axis[fi] = globalFft.freqs[fi] / 1e9
	}
	return axis
}

// GetFftLabels returns the component labels.
func GetFftLabels() []string {
	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()
	return globalFft.labels
}

// GetFftSpectrogram returns the spectrogram history.
func GetFftSpectrogram() ([][]float64, []float64) {
	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()
	return globalFft.spectrogramHistory, globalFft.spectrogramTimes
}

// GetFftSegmentProgress returns segment progress info:
// progress (0-1), segDuration (ns), elapsed (ns), totalSegments completed
func GetFftSegmentProgress() (float64, float64, float64, int) {
	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()

	if !globalFft.initialized || globalFft.segDuration <= 0 {
		return 0, 0, 0, 0
	}

	elapsed := Time - globalFft.segStart
	progress := elapsed / globalFft.segDuration
	if progress > 1 {
		progress = 1
	}

	return progress, globalFft.segDuration * 1e9, elapsed * 1e9, len(globalFft.spectrogramHistory)
}

// SetFftSpectrogramComponent sets which component to use for spectrogram.
func SetFftSpectrogramComponent(c int) {
	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()
	globalFft.spectrogramComp = c
	globalFft.spectrogramHistory = nil
	globalFft.spectrogramTimes = nil
}

// ClearFft resets all FFT accumulators.
func ClearFft() {
	globalFft.mu.Lock()
	defer globalFft.mu.Unlock()

	for c := 0; c < globalFft.nComp; c++ {
		for fi := range globalFft.accumReal[c] {
			globalFft.accumReal[c][fi] = 0
			globalFft.accumImag[c][fi] = 0
			globalFft.segReal[c][fi] = 0
			globalFft.segImag[c][fi] = 0
		}
		globalFft.signalBuf[c] = globalFft.signalBuf[c][:0]
		globalFft.baseline[c] = 0
	}
	globalFft.baselineSet = false
	globalFft.startTime = Time
	globalFft.segStart = Time
	globalFft.stepCounter = 0
	globalFft.spectrogramHistory = nil
	globalFft.spectrogramTimes = nil
}
