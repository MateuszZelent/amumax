package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/MathieuMoalic/amumax/src/engine"
	"github.com/MathieuMoalic/amumax/src/log"
)

// FftState holds the FFT data for WebSocket broadcast.
type FftState struct {
	ws *WebSocketManager

	Enabled              bool        `msgpack:"enabled"`
	FreqAxis             []float64   `msgpack:"freqAxis"`
	Labels               []string    `msgpack:"labels"`
	Spectrum             [][]float64 `msgpack:"spectrum"`
	Spectrogram          [][]float64 `msgpack:"spectrogram"`
	SpectrogramTimes     []float64   `msgpack:"spectrogramTimes"`
	SpectrogramComponent int         `msgpack:"spectrogramComponent"`
	SegProgress          float64     `msgpack:"segProgress"`
	SegDurationNs        float64     `msgpack:"segDurationNs"`
	SegElapsedNs         float64     `msgpack:"segElapsedNs"`
	TotalSegments        int         `msgpack:"totalSegments"`
}

func initFftAPI(e *echo.Group, ws *WebSocketManager) *FftState {
	fftState := &FftState{
		ws:      ws,
		Enabled: engine.FftEnabled,
	}

	e.POST("/api/fft/component", fftState.postFftComponent)
	e.POST("/api/fft/clear", fftState.postFftClear)

	return fftState
}

func (s *FftState) Update() {
	s.Enabled = engine.FftEnabled
	if !s.Enabled {
		return
	}

	s.FreqAxis = engine.GetFftFreqAxis()
	s.Labels = engine.GetFftLabels()
	s.Spectrum = engine.GetFftSpectrum()
	spectro, times := engine.GetFftSpectrogram()
	s.Spectrogram = spectro
	s.SpectrogramTimes = times
	s.SegProgress, s.SegDurationNs, s.SegElapsedNs, s.TotalSegments = engine.GetFftSegmentProgress()
}

func (s *FftState) postFftComponent(c echo.Context) error {
	type Request struct {
		Component int `msgpack:"component"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	engine.SetFftSpectrogramComponent(req.Component)
	s.SpectrogramComponent = req.Component
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func (s *FftState) postFftClear(c echo.Context) error {
	engine.ClearFft()
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}
