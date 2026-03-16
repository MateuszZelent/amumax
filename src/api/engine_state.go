package api

import "github.com/labstack/echo/v4"

type EngineState struct {
	Header    *HeaderState     `msgpack:"header"`
	Console   *ConsoleState    `msgpack:"console"`
	Preview   *PreviewState    `msgpack:"preview"`
	Solver    *SolverState     `msgpack:"solver"`
	Mesh      *MeshState       `msgpack:"mesh"`
	Params    *ParametersState `msgpack:"parameters"`
	TablePlot *TablePlotState  `msgpack:"tablePlot"`
	Metrics   *MetricsState    `msgpack:"metrics"`
	Fft       *FftState        `msgpack:"fft"`
}

type EngineStateNoPreview struct {
	Header    *HeaderState     `msgpack:"header"`
	Console   *ConsoleState    `msgpack:"console"`
	Solver    *SolverState     `msgpack:"solver"`
	Mesh      *MeshState       `msgpack:"mesh"`
	Params    *ParametersState `msgpack:"parameters"`
	TablePlot *TablePlotState  `msgpack:"tablePlot"`
	Metrics   *MetricsState    `msgpack:"metrics"`
	Fft       *FftState        `msgpack:"fft"`
}

func initEngineStateAPI(e *echo.Group, ws *WebSocketManager) *EngineState {
	return &EngineState{
		Header:    initHeaderAPI(),
		Console:   initConsoleAPI(e, ws),
		Preview:   initPreviewAPI(e, ws),
		Solver:    initSolverAPI(e, ws),
		Mesh:      initMeshAPI(e, ws),
		Params:    initParameterAPI(e, ws),
		TablePlot: initTablePlotAPI(e, ws),
		Metrics:   initMetricsAPI(e, ws),
		Fft:       initFftAPI(e, ws),
	}
}

func (es *EngineState) Update() {
	es.Header.Update()
	es.Console.Update()
	es.Preview.Update()
	es.Solver.Update()
	es.Mesh.Update()
	es.Params.Update()
	es.TablePlot.Update()
	es.Metrics.Update()
	es.Fft.Update()
}

func (es *EngineState) UpdateWithoutPreview() {
	es.Header.Update()
	es.Console.Update()
	es.Solver.Update()
	es.Mesh.Update()
	es.Params.Update()
	es.TablePlot.Update()
	es.Metrics.Update()
	es.Fft.Update()
}

func (es *EngineState) WithoutPreview() *EngineStateNoPreview {
	return &EngineStateNoPreview{
		Header:    es.Header,
		Console:   es.Console,
		Solver:    es.Solver,
		Mesh:      es.Mesh,
		Params:    es.Params,
		TablePlot: es.TablePlot,
		Metrics:   es.Metrics,
		Fft:       es.Fft,
	}
}
