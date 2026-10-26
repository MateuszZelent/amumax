package engine

import (
	"syscall"
	"testing"
)

func resetTerminationSignal(t *testing.T) {
	t.Helper()
	terminationSignal.Store(0)
	t.Cleanup(func() { terminationSignal.Store(0) })
}

func checkTerminationPanic(t *testing.T) any {
	t.Helper()
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		CheckTermination()
	}()
	if recovered == nil {
		t.Fatal("CheckTermination returned without controlled panic")
	}
	return recovered
}

func TestRequestTerminationKeepsFirstSignalAndMapsSIGTERMExit(t *testing.T) {
	resetTerminationSignal(t)

	RequestTermination(syscall.SIGTERM)
	RequestTermination(syscall.SIGINT)
	recovered := checkTerminationPanic(t)
	if code, ok := TerminationExitCode(recovered); !ok || code != 143 {
		t.Fatalf("TerminationExitCode(%T) = (%d, %v), want (143, true)", recovered, code, ok)
	}
}

func TestRequestTerminationMapsSIGINTExit(t *testing.T) {
	resetTerminationSignal(t)

	RequestTermination(syscall.SIGINT)
	recovered := checkTerminationPanic(t)
	if code, ok := TerminationExitCode(recovered); !ok || code != 130 {
		t.Fatalf("TerminationExitCode(%T) = (%d, %v), want (130, true)", recovered, code, ok)
	}
}

func TestCheckTerminationDoesNothingWithoutSignal(t *testing.T) {
	resetTerminationSignal(t)

	CheckTermination()
	if code, ok := TerminationExitCode("ordinary error"); ok || code != 0 {
		t.Fatalf("TerminationExitCode(non-termination) = (%d, %v), want (0, false)", code, ok)
	}
}
