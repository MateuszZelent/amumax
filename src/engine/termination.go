package engine

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
)

var terminationSignal atomic.Int32
var runFailure atomic.Bool

// RecordRunFailure marks an unhandled script or engine failure before cleanup.
func RecordRunFailure() { runFailure.Store(true) }

// RequestTermination is safe to call from a signal goroutine; the engine
// owner observes it at the next solver or script boundary.
func RequestTermination(sig os.Signal) {
	if value, ok := sig.(syscall.Signal); ok {
		terminationSignal.CompareAndSwap(0, int32(value))
	}
}

type terminationError struct{ signal int }

func (e terminationError) Error() string { return fmt.Sprintf("terminated by signal %d", e.signal) }

func CheckTermination() {
	if signal := terminationSignal.Load(); signal != 0 {
		panic(terminationError{signal: int(signal)})
	}
}

// TerminationExitCode identifies a controlled signal panic after cleanup.
func TerminationExitCode(value any) (int, bool) {
	err, ok := value.(terminationError)
	if !ok {
		return 0, false
	}
	return 128 + err.signal, true
}
