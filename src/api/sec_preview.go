package api

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/engine"
	"github.com/MathieuMoalic/amumax/src/log"
)

type PreviewState struct {
	ws                   *WebSocketManager
	globalQuantities     []string
	layerMask            [][]float32
	maskXSize            int
	maskYSize            int
	maskLayer            int
	Quantity             string       `msgpack:"quantity"`
	Unit                 string       `msgpack:"unit"`
	Component            string       `msgpack:"component"`
	Layer                int          `msgpack:"layer"`
	AllLayers            bool         `msgpack:"allLayers"`
	Type                 string       `msgpack:"type"`
	VectorFieldValues    []Vector3f   `msgpack:"vectorFieldValues"`
	VectorFieldPositions []Vector3i   `msgpack:"vectorFieldPositions"`
	ScalarField          [][3]float32 `msgpack:"scalarField"`
	Min                  float32      `msgpack:"min"`
	Max                  float32      `msgpack:"max"`
	Refresh              bool         `msgpack:"refresh"`
	NComp                int          `msgpack:"nComp"`

	MaxPoints             int    `msgpack:"maxPoints"`
	DataPointsCount       int    `msgpack:"dataPointsCount"`
	XPossibleSizes        []int  `msgpack:"xPossibleSizes"`
	YPossibleSizes        []int  `msgpack:"yPossibleSizes"`
	XChosenSize           int    `msgpack:"xChosenSize"`
	YChosenSize           int    `msgpack:"yChosenSize"`
	AppliedXChosenSize    int    `msgpack:"appliedXChosenSize"`
	AppliedYChosenSize    int    `msgpack:"appliedYChosenSize"`
	AppliedLayerStride    int    `msgpack:"appliedLayerStride"`
	AutoDownscaled        bool   `msgpack:"autoDownscaled"`
	AutoDownscaleMessage  string `msgpack:"autoDownscaleMessage"`
}

type Vector3f struct {
	X float32 `msgpack:"x"`
	Y float32 `msgpack:"y"`
	Z float32 `msgpack:"z"`
}

type Vector3i struct {
	X int `msgpack:"x"`
	Y int `msgpack:"y"`
	Z int `msgpack:"z"`
}

func initPreviewAPI(e *echo.Group, ws *WebSocketManager) *PreviewState {
	previewState := &PreviewState{
		Quantity:             "m",
		Component:            "3D",
		Layer:                0,
		AllLayers:            false,
		MaxPoints:            131072,
		Type:                 "3D",
		VectorFieldValues:    nil,
		VectorFieldPositions: nil,
		ScalarField:          nil,
		Min:                  0,
		Max:                  0,
		Refresh:              true,
		NComp:                3,
		DataPointsCount:      0,
		XPossibleSizes:       nil,
		YPossibleSizes:       nil,
		XChosenSize:          engine.Mesh.Nx,
		YChosenSize:          engine.Mesh.Ny,
		ws:                   ws,
		globalQuantities:     []string{"B_demag", "B_ext", "B_eff", "B_oersted", "Edens_demag", "Edens_ext", "Edens_eff", "geom", "SpongeAlpha"},
	}
	previewState.addPossibleDownscaleSizes()
	previewState.AppliedXChosenSize = previewState.XChosenSize
	previewState.AppliedYChosenSize = previewState.YChosenSize
	previewState.AppliedLayerStride = 1
	e.POST("/api/preview/component", previewState.postPreviewComponent)
	e.POST("/api/preview/quantity", previewState.postPreviewQuantity)
	e.POST("/api/preview/layer", previewState.postPreviewLayer)
	e.POST("/api/preview/maxpoints", previewState.postPreviewMaxPoints)
	e.POST("/api/preview/refresh", previewState.postPreviewRefresh)
	e.POST("/api/preview/XChosenSize", previewState.postXChosenSize)
	e.POST("/api/preview/YChosenSize", previewState.postYChosenSize)
	e.POST("/api/preview/allLayers", previewState.postAllLayers)

	return previewState
}

func (s *PreviewState) getQuantity() engine.Quantity {
	quantity, exists := engine.Quantities[s.Quantity]
	if !exists {
		log.Log.Err("Quantity not found: %v", s.Quantity)
	}
	return quantity
}

func (s *PreviewState) getComponent() int {
	return compStringToIndex(s.Component)
}

func (s *PreviewState) Update() {
	engine.InjectAndWait(s.UpdateQuantityBuffer)
}

type previewSizing struct {
	RequestedX         int
	RequestedY         int
	AppliedX           int
	AppliedY           int
	RequestedDepth     int
	AppliedDepth       int
	LayerStride        int
	RequestedPoints    int
	AppliedPoints      int
	AutoDownscaled     bool
}

func (s *PreviewState) UpdateQuantityBuffer() {
	defer func() {
		if r := recover(); r != nil {
			log.Log.Warn("Recovered from panic in UpdateQuantityBuffer: %v", r)
			s.ScalarField = nil
			s.VectorFieldPositions = nil
			s.VectorFieldValues = nil
			s.DataPointsCount = 0
		}
	}()

	if s.XChosenSize == 0 || s.YChosenSize == 0 {
		log.Log.Debug("XChosenSize or YChosenSize is 0")
		return
	}

	componentCount := 1
	if s.Type == "3D" {
		componentCount = 3
	}
	GPUIn := engine.ValueOf(s.getQuantity())
	defer cuda.Recycle(GPUIn)

	depthLayers := 1
	if s.AllLayers && s.Type == "3D" {
		depthLayers = maxInt(GPUIn.Size()[2], 1)
	}
	sizing := s.resolvePreviewSizing(depthLayers)
	s.applyResolvedSizing(sizing)

	if sizing.AppliedX == 0 || sizing.AppliedY == 0 {
		log.Log.Debug("Applied preview size is 0")
		return
	}

	if s.AllLayers && s.Type == "3D" {
		s.updateAllLayers(GPUIn, componentCount, sizing.LayerStride)
		return
	}

	if s.AllLayers && s.Type != "3D" {
		s.updateAllLayersScalar(GPUIn)
		return
	}

	CPUOut := data.NewSlice(componentCount, [3]int{sizing.AppliedX, sizing.AppliedY, 1})
	GPUOut := cuda.NewSlice(1, [3]int{sizing.AppliedX, sizing.AppliedY, 1})
	defer GPUOut.Free()

	if s.Type == "3D" {
		for c := 0; c < componentCount; c++ {
			cuda.Resize(GPUOut, GPUIn.Comp(c), s.Layer)
			data.Copy(CPUOut.Comp(c), GPUOut)
		}
		s.normalizeVectors(CPUOut)
		s.UpdateVectorField(CPUOut.Vectors())
		return
	}

	s.ensureMask(sizing.AppliedX, sizing.AppliedY)
	if s.getQuantity().NComp() > 1 {
		cuda.Resize(GPUOut, GPUIn.Comp(s.getComponent()), s.Layer)
		data.Copy(CPUOut.Comp(0), GPUOut)
	} else {
		cuda.Resize(GPUOut, GPUIn.Comp(0), s.Layer)
		data.Copy(CPUOut.Comp(0), GPUOut)
	}
	s.UpdateScalarField(CPUOut.Scalars())
}

func (s *PreviewState) normalizeVectors(f *data.Slice) {
	a := f.Vectors()
	maxnorm := 0.0
	for i := range a[0] {
		for j := range a[0][i] {
			for k := range a[0][i][j] {
				x, y, z := a[0][i][j][k], a[1][i][j][k], a[2][i][j][k]
				norm := math.Sqrt(float64(x*x + y*y + z*z))
				if norm > maxnorm {
					maxnorm = norm
				}
			}
		}
	}
	if maxnorm == 0 {
		return
	}
	factor := float32(1 / maxnorm)

	for i := range a[0] {
		for j := range a[0][i] {
			for k := range a[0][i][j] {
				a[0][i][j][k] *= factor
				a[1][i][j][k] *= factor
				a[2][i][j][k] *= factor
			}
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ceilDiv(n, d int) int {
	if d <= 0 {
		return 0
	}
	return (n + d - 1) / d
}

func floorAllowedSize(arr []int, target int) int {
	if target <= 1 || len(arr) == 0 {
		return maxInt(target, 1)
	}
	best := 1
	for _, value := range arr {
		if value > target {
			break
		}
		best = value
	}
	return best
}

func previousAllowedSize(arr []int, current int) int {
	if current <= 1 || len(arr) == 0 {
		return 1
	}
	prev := 1
	for _, value := range arr {
		if value >= current {
			break
		}
		prev = value
	}
	return prev
}

func (s *PreviewState) resolvePreviewSizing(depthLayers int) previewSizing {
	requestedX := maxInt(s.XChosenSize, 1)
	requestedY := maxInt(s.YChosenSize, 1)
	if len(s.XPossibleSizes) > 0 && !containsInt(s.XPossibleSizes, requestedX) {
		requestedX = closestInArray(s.XPossibleSizes, requestedX)
	}
	if len(s.YPossibleSizes) > 0 && !containsInt(s.YPossibleSizes, requestedY) {
		requestedY = closestInArray(s.YPossibleSizes, requestedY)
	}

	sizing := previewSizing{
		RequestedX:     requestedX,
		RequestedY:     requestedY,
		AppliedX:       requestedX,
		AppliedY:       requestedY,
		RequestedDepth: maxInt(depthLayers, 1),
		AppliedDepth:   maxInt(depthLayers, 1),
		LayerStride:    1,
	}
	sizing.RequestedPoints = sizing.RequestedX * sizing.RequestedY * sizing.RequestedDepth

	maxPoints := maxInt(s.MaxPoints, 8)
	if sizing.RequestedPoints <= maxPoints {
		sizing.AppliedPoints = sizing.RequestedPoints
		return sizing
	}

	scale := math.Sqrt(float64(maxPoints) / float64(sizing.RequestedPoints))
	targetX := maxInt(1, int(math.Floor(float64(sizing.RequestedX)*scale)))
	targetY := maxInt(1, int(math.Floor(float64(sizing.RequestedY)*scale)))
	sizing.AppliedX = floorAllowedSize(s.XPossibleSizes, targetX)
	sizing.AppliedY = floorAllowedSize(s.YPossibleSizes, targetY)
	if sizing.AppliedX > sizing.RequestedX {
		sizing.AppliedX = sizing.RequestedX
	}
	if sizing.AppliedY > sizing.RequestedY {
		sizing.AppliedY = sizing.RequestedY
	}

	for {
		sizing.AppliedDepth = ceilDiv(sizing.RequestedDepth, sizing.LayerStride)
		sizing.AppliedPoints = sizing.AppliedX * sizing.AppliedY * sizing.AppliedDepth
		if sizing.AppliedPoints <= maxPoints {
			break
		}

		reduced := false
		if sizing.AppliedX >= sizing.AppliedY && sizing.AppliedX > 1 {
			next := previousAllowedSize(s.XPossibleSizes, sizing.AppliedX)
			if next < sizing.AppliedX {
				sizing.AppliedX = next
				reduced = true
			}
		}
		if !reduced && sizing.AppliedY > 1 {
			next := previousAllowedSize(s.YPossibleSizes, sizing.AppliedY)
			if next < sizing.AppliedY {
				sizing.AppliedY = next
				reduced = true
			}
		}
		if !reduced && sizing.RequestedDepth > 1 && sizing.AppliedDepth > 1 {
			sizing.LayerStride++
			reduced = true
		}
		if !reduced {
			break
		}
	}

	sizing.AppliedDepth = ceilDiv(sizing.RequestedDepth, sizing.LayerStride)
	sizing.AppliedPoints = sizing.AppliedX * sizing.AppliedY * sizing.AppliedDepth
	sizing.AutoDownscaled = sizing.AppliedX != sizing.RequestedX || sizing.AppliedY != sizing.RequestedY || sizing.LayerStride != 1
	return sizing
}

func (s *PreviewState) applyResolvedSizing(sizing previewSizing) {
	s.AppliedXChosenSize = sizing.AppliedX
	s.AppliedYChosenSize = sizing.AppliedY
	s.AppliedLayerStride = sizing.LayerStride
	s.AutoDownscaled = sizing.AutoDownscaled
	if !sizing.AutoDownscaled {
		s.AutoDownscaleMessage = ""
		return
	}

	requestedShape := fmt.Sprintf("%dx%d", sizing.RequestedX, sizing.RequestedY)
	appliedShape := fmt.Sprintf("%dx%d", sizing.AppliedX, sizing.AppliedY)
	if sizing.RequestedDepth > 1 {
		requestedShape = fmt.Sprintf("%s x %d", requestedShape, sizing.RequestedDepth)
		appliedShape = fmt.Sprintf("%s x %d", appliedShape, sizing.AppliedDepth)
	}
	message := fmt.Sprintf("Preview auto-scaled from %s to %s to stay within %d points", requestedShape, appliedShape, maxInt(s.MaxPoints, 8))
	if sizing.LayerStride > 1 {
		message = fmt.Sprintf("%s (sampling every %d layer)", message, sizing.LayerStride)
	}
	s.AutoDownscaleMessage = message
}

func (s *PreviewState) updateAllLayers(GPUIn *data.Slice, componentCount int, layerStride int) {
	nz := GPUIn.Size()[2]
	valArray := make([]Vector3f, 0)
	posArray := make([]Vector3i, 0)

	xSize := maxInt(s.AppliedXChosenSize, 1)
	ySize := maxInt(s.AppliedYChosenSize, 1)
	CPUOut := data.NewSlice(componentCount, [3]int{xSize, ySize, 1})
	GPUOut := cuda.NewSlice(1, [3]int{xSize, ySize, 1})
	defer GPUOut.Free()

	for layer := 0; layer < nz; layer += maxInt(layerStride, 1) {
		for c := 0; c < componentCount; c++ {
			cuda.Resize(GPUOut, GPUIn.Comp(c), layer)
			data.Copy(CPUOut.Comp(c), GPUOut)
		}
		vf := CPUOut.Vectors()
		yLen := len(vf[0][0])
		xLen := len(vf[0][0][0])
		for posx := 0; posx < xLen; posx++ {
			for posy := 0; posy < yLen; posy++ {
				valx := vf[0][0][posy][posx]
				valy := vf[1][0][posy][posx]
				valz := vf[2][0][posy][posx]
				if (valx == 0 && valy == 0 && valz == 0) || math.IsNaN(float64(valx)) {
					continue
				}
				posArray = append(posArray, Vector3i{X: posx, Y: posy, Z: layer})
				valArray = append(valArray, Vector3f{X: valx, Y: valy, Z: valz})
			}
		}
	}

	maxnorm := float64(0)
	for _, v := range valArray {
		norm := math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z))
		if norm > maxnorm {
			maxnorm = norm
		}
	}
	if maxnorm > 0 {
		factor := float32(1 / maxnorm)
		for i := range valArray {
			valArray[i].X *= factor
			valArray[i].Y *= factor
			valArray[i].Z *= factor
		}
	}

	s.VectorFieldPositions = posArray
	s.VectorFieldValues = valArray
	s.ScalarField = nil
	s.DataPointsCount = len(valArray)
}

func (s *PreviewState) updateAllLayersScalar(GPUIn *data.Slice) {
	nz := GPUIn.Size()[2]
	xSize := maxInt(s.AppliedXChosenSize, 1)
	ySize := maxInt(s.AppliedYChosenSize, 1)

	acc := make([][]float32, ySize)
	for y := 0; y < ySize; y++ {
		acc[y] = make([]float32, xSize)
	}
	hasValue := make([][]bool, ySize)
	for y := 0; y < ySize; y++ {
		hasValue[y] = make([]bool, xSize)
	}

	CPUOut := data.NewSlice(1, [3]int{xSize, ySize, 1})
	GPUOut := cuda.NewSlice(1, [3]int{xSize, ySize, 1})
	defer GPUOut.Free()

	compIdx := 0
	if s.getQuantity().NComp() > 1 {
		compIdx = s.getComponent()
	}

	for layer := 0; layer < nz; layer++ {
		cuda.Resize(GPUOut, GPUIn.Comp(compIdx), layer)
		data.Copy(CPUOut.Comp(0), GPUOut)
		scalars := CPUOut.Scalars()

		for posy := 0; posy < ySize; posy++ {
			for posx := 0; posx < xSize; posx++ {
				val := scalars[0][posy][posx]
				if math.IsNaN(float64(val)) {
					continue
				}
				if !hasValue[posy][posx] {
					acc[posy][posx] = val
					hasValue[posy][posx] = true
				} else if math.Abs(float64(val)) > math.Abs(float64(acc[posy][posx])) {
					acc[posy][posx] = val
				}
			}
		}
	}

	var min, max float32
	minMaxSet := false
	valArray := make([][3]float32, 0, xSize*ySize)

	for posx := 0; posx < xSize; posx++ {
		for posy := 0; posy < ySize; posy++ {
			if !hasValue[posy][posx] {
				continue
			}
			val := acc[posy][posx]
			if !minMaxSet {
				min, max = val, val
				minMaxSet = true
			} else {
				if val < min {
					min = val
				}
				if val > max {
					max = val
				}
			}
			valArray = append(valArray, [3]float32{float32(posx), float32(posy), val})
		}
	}

	if len(valArray) == 0 {
		s.Min = 0
		s.Max = 0
		s.ScalarField = nil
		s.VectorFieldValues = nil
		s.VectorFieldPositions = nil
		s.DataPointsCount = 0
		return
	}

	s.Min = min
	s.Max = max
	s.ScalarField = valArray
	s.VectorFieldValues = nil
	s.VectorFieldPositions = nil
	s.DataPointsCount = len(valArray)
}

func (s *PreviewState) UpdateVectorField(vectorField [3][][][]float32) {
	yLen := len(vectorField[0][0])
	xLen := len(vectorField[0][0][0])
	maxCount := xLen * yLen

	valArray := make([]Vector3f, 0, maxCount)
	posArray := make([]Vector3i, 0, maxCount)
	for posx := 0; posx < xLen; posx++ {
		for posy := 0; posy < yLen; posy++ {
			valx := vectorField[0][0][posy][posx]
			valy := vectorField[1][0][posy][posx]
			valz := vectorField[2][0][posy][posx]
			if (valx == 0 && valy == 0 && valz == 0) || math.IsNaN(float64(valx)) {
				continue
			}
			posArray = append(posArray, Vector3i{X: posx, Y: posy, Z: 0})
			valArray = append(valArray, Vector3f{X: valx, Y: valy, Z: valz})
		}
	}
	s.VectorFieldPositions = posArray
	s.VectorFieldValues = valArray
	s.ScalarField = nil
	s.DataPointsCount = len(valArray)
}

func (s *PreviewState) UpdateScalarField(scalarField [][][]float32) {
	xLen := len(scalarField[0][0])
	yLen := len(scalarField[0])
	min, max := float32(0), float32(0)
	hasValue := false

	valArray := make([][3]float32, 0, xLen*yLen)
	for posx := 0; posx < xLen; posx++ {
		for posy := 0; posy < yLen; posy++ {
			if !contains(s.globalQuantities, s.Quantity) && s.layerMask != nil && s.layerMask[posy][posx] == 0 {
				continue
			}
			val := scalarField[0][posy][posx]
			if !hasValue {
				min, max = val, val
				hasValue = true
			} else {
				if val < min {
					min = val
				}
				if val > max {
					max = val
				}
			}
			valArray = append(valArray, [3]float32{float32(posx), float32(posy), val})
		}
	}
	if len(valArray) == 0 {
		log.Log.Warn("No data in scalar field")
		s.Min = 0
		s.Max = 0
		s.ScalarField = nil
		s.VectorFieldValues = nil
		s.VectorFieldPositions = nil
		s.DataPointsCount = 0
		return
	}

	s.Min = min
	s.Max = max
	s.ScalarField = valArray
	s.VectorFieldValues = nil
	s.VectorFieldPositions = nil
	s.DataPointsCount = len(valArray)
}

func (s *PreviewState) ensureMask(xSize, ySize int) {
	if s.layerMask != nil && s.maskXSize == xSize && s.maskYSize == ySize && s.maskLayer == s.Layer {
		return
	}
	s.updateMaskForSize(xSize, ySize)
}

func (s *PreviewState) updateMask() {
	s.updateMaskForSize(s.XChosenSize, s.YChosenSize)
}

func (s *PreviewState) updateMaskForSize(xSize, ySize int) {
	defer func() {
		if r := recover(); r != nil {
			log.Log.Warn("Recovered from panic in updateMask: %v", r)
			s.layerMask = nil
			s.maskXSize = 0
			s.maskYSize = 0
			s.maskLayer = 0
		}
	}()
	if xSize == 0 || ySize == 0 {
		log.Log.Debug("XChosenSize or YChosenSize is 0")
		return
	}

	geom := engine.Geometry
	GPUFullsize := cuda.Buffer(geom.NComp(), geom.Buffer.Size())
	geom.EvalTo(GPUFullsize)
	defer cuda.Recycle(GPUFullsize)

	GPUResized := cuda.NewSlice(1, [3]int{xSize, ySize, 1})
	defer GPUResized.Free()
	cuda.Resize(GPUResized, GPUFullsize.Comp(0), s.Layer)

	CPUOut := data.NewSlice(1, [3]int{xSize, ySize, 1})
	defer CPUOut.Free()
	data.Copy(CPUOut.Comp(0), GPUResized)

	s.layerMask = CPUOut.Scalars()[0]
	s.maskXSize = xSize
	s.maskYSize = ySize
	s.maskLayer = s.Layer
}

func contains(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}

func closestInArray(arr []int, target int) int {
	closest := arr[0]
	minDiff := math.Abs(float64(target - closest))

	for _, num := range arr {
		diff := math.Abs(float64(target - num))
		if diff < minDiff {
			minDiff = diff
			closest = num
		}
	}

	return closest
}

func compStringToIndex(comp string) int {
	switch comp {
	case "x":
		return 0
	case "y":
		return 1
	case "z":
		return 2
	case "3D":
		return -1
	case "None":
		return 0
	}
	log.Log.ErrAndExit("Invalid component string")
	return -2
}

// A valid destination size is a positive integer less than or equal to srcsize that evenly divides srcsize.
func (s *PreviewState) addPossibleDownscaleSizes() {
	// retry until engine.Mesh.Nx and engine.Mesh.Ny are not 0
	for engine.Mesh.Nx == 0 || engine.Mesh.Ny == 0 {
		time.Sleep(1 * time.Second)
	}
	if engine.Mesh.Nx == 0 || engine.Mesh.Ny == 0 {
		log.Log.Err("Nx or Ny is 0")
	}
	// iterate over engine.Mesh.Nx and engine.Mesh.Ny
	for dstsize := 1; dstsize <= engine.Mesh.Nx; dstsize++ {
		if engine.Mesh.Nx%dstsize == 0 {
			s.XPossibleSizes = append(s.XPossibleSizes, dstsize)
		}
	}
	for dstsize := 1; dstsize <= engine.Mesh.Ny; dstsize++ {
		if engine.Mesh.Ny%dstsize == 0 {
			s.YPossibleSizes = append(s.YPossibleSizes, dstsize)
		}
	}
	if len(s.YPossibleSizes) == 0 || len(s.XPossibleSizes) == 0 {
		log.Log.Err("No possible sizes found")
	}
	if engine.PreviewXDataPoints != 0 {
		s.XChosenSize = closestInArray(s.XPossibleSizes, engine.PreviewXDataPoints)
	} else {
		s.XChosenSize = closestInArray(s.XPossibleSizes, 100)
	}
	if engine.PreviewYDataPoints != 0 {
		s.YChosenSize = closestInArray(s.YPossibleSizes, engine.PreviewYDataPoints)
	} else {
		s.YChosenSize = closestInArray(s.YPossibleSizes, 100)
	}
}

func (s *PreviewState) updatePreviewType() {
	var fieldType string
	isVectorField := s.NComp == 3 && s.getComponent() == -1
	if isVectorField {
		fieldType = "3D"
	} else {
		fieldType = "2D"
	}
	if fieldType != s.Type {
		s.Type = fieldType
		s.Refresh = true
	}
}

func (s *PreviewState) validateComponent() {
	s.NComp = s.getQuantity().NComp()
	switch s.NComp {
	case 1:
		s.Component = "None"
	case 3:
		if s.Component == "None" {
			s.Component = "3D"
		}
	default:
		log.Log.Err("Invalid number of components")
		// reset to default
		s.Quantity = "m"
		s.Component = "3D"
	}
}

func (s *PreviewState) postPreviewComponent(c echo.Context) error {
	type Request struct {
		Component string `msgpack:"component"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	s.Component = req.Component
	s.validateComponent()
	s.updatePreviewType()
	s.Refresh = true
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func (s *PreviewState) postPreviewQuantity(c echo.Context) error {
	type Request struct {
		Quantity string `msgpack:"quantity"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	_, exists := engine.Quantities[req.Quantity]
	if !exists {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Quantity not found"})
	}
	s.Quantity = req.Quantity
	s.validateComponent()
	s.Refresh = true
	s.updatePreviewType()
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func (s *PreviewState) postPreviewLayer(c echo.Context) error {
	type Request struct {
		Layer int `msgpack:"layer"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}

	s.Layer = req.Layer
	s.Refresh = true
	engine.InjectAndWait(s.updateMask)
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func (s *PreviewState) postPreviewMaxPoints(c echo.Context) error {
	type Request struct {
		MaxPoints int `msgpack:"maxPoints"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	if req.MaxPoints < 8 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "MaxPoints must be at least 8"})
	}
	s.MaxPoints = req.MaxPoints
	s.Refresh = true
	engine.InjectAndWait(s.updateMask)
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func (s *PreviewState) postPreviewRefresh(c echo.Context) error {
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func containsInt(arr []int, target int) bool {
	for _, v := range arr {
		if v == target {
			return true
		}
	}
	return false
}

func (s *PreviewState) postXChosenSize(c echo.Context) error {
	type Request struct {
		XChosenSize int `msgpack:"xChosenSize"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	if !containsInt(s.XPossibleSizes, req.XChosenSize) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid xChosenSize"})
	}
	s.XChosenSize = req.XChosenSize
	s.Refresh = true
	engine.InjectAndWait(s.updateMask)
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func (s *PreviewState) postYChosenSize(c echo.Context) error {
	type Request struct {
		YChosenSize int `msgpack:"yChosenSize"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	if !containsInt(s.YPossibleSizes, req.YChosenSize) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid yChosenSize"})
	}
	s.YChosenSize = req.YChosenSize
	s.Refresh = true
	engine.InjectAndWait(s.updateMask)
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}

func (s *PreviewState) postAllLayers(c echo.Context) error {
	type Request struct {
		AllLayers bool `msgpack:"allLayers"`
	}
	req := new(Request)
	if err := c.Bind(req); err != nil {
		log.Log.Err("%v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}
	s.AllLayers = req.AllLayers
	s.Refresh = true
	s.ws.broadcastEngineState()
	return c.JSON(http.StatusOK, nil)
}
