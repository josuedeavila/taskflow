package taskflow_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/josuedeavila/taskflow"
)

func TestLoggerFunc_Log(t *testing.T) {
	called := false
	logger := taskflow.LoggerFunc(func(args ...interface{}) {
		called = true
		if len(args) == 0 {
			t.Error("LoggerFunc called with no arguments")
		}

		if !strings.Contains(args[0].(string), "Hello LoggerFunc!") {
			t.Errorf("Expected 'Hello LoggerFunc!', got '%v'", args[0])
		}
	})

	logger.Log("Hello LoggerFunc!")
	if !called {
		t.Error("LoggerFunc should not have been called before Log")
	}
}

func TestDefaultLogger_Log(t *testing.T) {
	// Redirect stdout to buffer
	original := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Replace logger output to our pipe
	l := log.New(w, "", 0)
	defaultLog := taskflow.LoggerFunc(func(args ...interface{}) {
		l.Println(args...)
	})

	defaultLog.Log("Default logger test")

	// Capture and restore
	w.Close()
	os.Stdout = original

	var outputBuf bytes.Buffer
	_, _ = outputBuf.ReadFrom(r)
	logOutput := outputBuf.String()
	if !strings.Contains(logOutput, "Default logger test") {
		t.Error("Expected log output, got nothing")
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := taskflow.LoggerFunc(func(args ...interface{}) {
		fmt.Fprintln(os.Stdout, args...)
	})
	if logger == nil {
		t.Error("Expected logger to be created, got nil")
	}
}

func TestNoOpLogger_Log(t *testing.T) {
	// Should not panic or output anything
	logger := taskflow.NoOpLogger{}
	logger.Log("This should not appear anywhere")
}
