package engine

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/timer"
)

// Asynchronous I/O queue flushes data to disk while simulation keeps running.
// See save.go, autosave.go

var (
	saveQue       chan func() // passes save requests to runSaver for asyc IO
	queLen        atom        // # tasks in queue
	outputFailure struct {
		sync.Mutex
		err error
	}
)

func init() {
	saveQue = make(chan func())
	go runSaver()
}

// Atomic int
type atom int32

func (a *atom) Add(v int32) {
	atomic.AddInt32((*int32)(a), v)
}

func (a *atom) Load() int32 {
	return atomic.LoadInt32((*int32)(a))
}

func queOutput(f func()) {
	checkOutputError()
	if cuda.Synchronous {
		timer.Start("io")
	}
	queLen.Add(1)
	saveQue <- f
	if cuda.Synchronous {
		timer.Stop("io")
	}
}

// Continuously executes tasks the from SaveQue channel.
func runSaver() {
	for f := range saveQue {
		func() {
			defer queLen.Add(-1)
			defer captureOutputPanic()
			f()
		}()
	}
}

func captureOutputPanic() {
	if value := recover(); value != nil {
		outputFailure.Lock()
		if outputFailure.err == nil {
			outputFailure.err = fmt.Errorf("asynchronous output: %v", value)
		}
		outputFailure.Unlock()
	}
}

func outputError() error {
	outputFailure.Lock()
	defer outputFailure.Unlock()
	return outputFailure.err
}

func checkOutputError() {
	if err := outputError(); err != nil {
		panic(err)
	}
}

// Finalizer function called upon program exit.
// Waits until all asynchronous output has been saved.
func drainOutput() {
	waitOutput()
	checkOutputError()
}

func waitOutput() {
	if saveQue == nil {
		return
	}
	for queLen.Load() > 0 {
		time.Sleep(time.Millisecond)
	}
}
