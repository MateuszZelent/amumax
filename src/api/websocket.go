package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/MathieuMoalic/amumax/src/engine"
	"github.com/MathieuMoalic/amumax/src/log"
)

// WebSocketManager holds state previously stored in global variables.
type WebSocketManager struct {
	upgrader              websocket.Upgrader
	connections           *connectionManager
	previewConnections    *connectionManager
	lastStep              int
	lastMainBroadcast     time.Time
	mainBroadcastInterval time.Duration
	broadcastStop         chan struct{}
	broadcastStart        sync.Once
	engineState           *EngineState
}

type connectionManager struct {
	conns map[*websocket.Conn]struct{}
	mu    sync.Mutex
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		conns: make(map[*websocket.Conn]struct{}),
		mu:    sync.Mutex{},
	}
}

func newWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		upgrader: websocket.Upgrader{
			EnableCompression: true,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connections:           newConnectionManager(),
		previewConnections:    newConnectionManager(),
		mainBroadcastInterval: 5 * time.Second,
		broadcastStop:         make(chan struct{}),
	}
}

func (cm *connectionManager) add(ws *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.conns[ws] = struct{}{}
}

func (cm *connectionManager) remove(ws *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.conns, ws)
}

func (cm *connectionManager) count() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.conns)
}

func (cm *connectionManager) broadcast(msg []byte) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for ws := range cm.conns {
		err := ws.WriteMessage(websocket.BinaryMessage, msg)
		if err != nil {
			log.Log.Err("Error sending message via WebSocket: %v", err)
			if cerr := ws.Close(); cerr != nil {
				log.Log.Err("Error closing WebSocket: %v", cerr)
			}
			delete(cm.conns, ws)
		}
	}
}

func (wsManager *WebSocketManager) websocketEntrypoint(c echo.Context) error {
	return wsManager.websocketEntrypointFor(c, wsManager.connections, "main", wsManager.broadcastEngineState)
}

func (wsManager *WebSocketManager) websocketPreviewEntrypoint(c echo.Context) error {
	return wsManager.websocketEntrypointFor(c, wsManager.previewConnections, "preview", wsManager.broadcastPreviewState)
}

func (wsManager *WebSocketManager) websocketEntrypointFor(c echo.Context, cm *connectionManager, name string, onConnect func()) error {
	log.Log.Debug("New %s WebSocket connection, upgrading...", name)
	ws, err := wsManager.upgrader.Upgrade(c.Response(), c.Request(), nil)
	log.Log.Debug("New %s WebSocket connection upgraded", name)
	if err != nil {
		log.Log.Err("Error upgrading %s websocket connection: %v", name, err)
		return err
	}
	ws.EnableWriteCompression(true)
	defer func() {
		if err := ws.Close(); err != nil {
			log.Log.Err("Error closing %s websocket: %v", name, err)
		}
	}()

	cm.add(ws)
	defer cm.remove(ws)
	wsManager.engineState.Preview.Refresh = true
	onConnect()

	// Channel to signal when to stop the goroutine
	done := make(chan struct{})
	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
		log.Log.Debug("%s websocket connection closed by client", name)
		return nil
	case <-wsManager.broadcastStop:
		return nil
	}
}

func (wsManager *WebSocketManager) broadcastEngineState() {
	wsManager.engineState.Update()
	msg, err := msgpack.Marshal(wsManager.engineState)
	if err != nil {
		log.Log.Err("Error marshaling combined message: %v", err)
		return
	}
	wsManager.connections.broadcast(msg)
	// Reset the refresh flag
	wsManager.engineState.Preview.Refresh = false
}

func (wsManager *WebSocketManager) broadcastPreviewState() {
	if wsManager.engineState == nil || wsManager.engineState.Preview == nil {
		return
	}
	wsManager.engineState.Preview.Update()
	msg, err := msgpack.Marshal(wsManager.engineState.Preview)
	if err != nil {
		log.Log.Err("Error marshaling preview message: %v", err)
		return
	}
	wsManager.previewConnections.broadcast(msg)
	wsManager.engineState.Preview.Refresh = false
}

func (wsManager *WebSocketManager) broadcastEngineStateWithoutPreview() {
	if wsManager.engineState == nil {
		return
	}
	wsManager.engineState.UpdateWithoutPreview()
	msg, err := msgpack.Marshal(wsManager.engineState.WithoutPreview())
	if err != nil {
		log.Log.Err("Error marshaling non-preview message: %v", err)
		return
	}
	wsManager.connections.broadcast(msg)
}

func (wsManager *WebSocketManager) startBroadcastLoop() {
	wsManager.broadcastStart.Do(func() {
		go func() {
			for {
				select {
				case <-wsManager.broadcastStop:
					return
				default:
					if engine.NSteps != wsManager.lastStep {
						if wsManager.previewConnections.count() > 0 {
							wsManager.broadcastPreviewState()
						}
						if wsManager.connections.count() > 0 {
							now := time.Now()
							if wsManager.lastMainBroadcast.IsZero() || now.Sub(wsManager.lastMainBroadcast) >= wsManager.mainBroadcastInterval {
								wsManager.broadcastEngineStateWithoutPreview()
								wsManager.lastMainBroadcast = now
							}
						}
						wsManager.lastStep = engine.NSteps
					}
					time.Sleep(1 * time.Second)
				}
			}
		}()
	})
}
