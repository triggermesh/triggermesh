/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trace_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/devigned/tab"
	. "github.com/triggermesh/triggermesh/pkg/sources/adapter/azureeventhubssource/trace"
)

func TestNoOpTracerWithLogger_Logger(t *testing.T) {
	// reset default tracer after the test completes
	t.Cleanup(func() {
		tab.Register(&tab.NoOpTracer{})
	})

	// span attributes that we expect the logger to propagate to log entries
	attrs := []tab.Attribute{
		{Key: "somestring", Value: "xyz"},
		{Key: "someint", Value: 42},
		{Key: "somebool", Value: false},
	}

	ctx := context.Background()

	t.Run("info level", func(t *testing.T) {
		var logOutput bytes.Buffer
		tracer := newLogTracer(&logOutput)
		logger := tracer.FromContext(ctx).Logger()

		logger.Info("test message", attrs...)

		const expect = `{"level":"info","msg":"test message","somestring":"xyz","someint":42,"somebool":false}` + "\n"

		if got := logOutput.String(); got != expect {
			t.Errorf("Expected\n  %sgot\n  %s", expect, got)
		}
	})

	t.Run("error level", func(t *testing.T) {
		var logOutput bytes.Buffer
		tracer := newLogTracer(&logOutput)
		logger := tracer.FromContext(ctx).Logger()

		logger.Error(fmt.Errorf("test message"), attrs...)

		const expect = `{"level":"error","msg":"test message","somestring":"xyz","someint":42,"somebool":false}` + "\n"

		if got := logOutput.String(); got != expect {
			t.Errorf("Expected\n  %sgot\n  %s", expect, got)
		}
	})

	t.Run("fatal level", func(t *testing.T) {
		var logOutput bytes.Buffer
		tracer := newLogTracer(&logOutput,
			// zap calls os.Exit(1) by default on Fatal, which is
			// difficult to handle in tests. WriteThenNoop is
			// unfortunately explicitly rejected by zap on Fatal,
			// so we fallback to WriteThenPanic and call recover().
			zap.WithFatalHook(zapcore.WriteThenPanic),
		)
		logger := tracer.FromContext(ctx).Logger()

		func() {
			defer func() { _ = recover() }()
			logger.Fatal("test message", attrs...)
		}()

		const expect = `{"level":"fatal","msg":"test message","somestring":"xyz","someint":42,"somebool":false}` + "\n"

		if got := logOutput.String(); got != expect {
			t.Errorf("Expected\n  %sgot\n  %s", expect, got)
		}
	})

	t.Run("debug level", func(t *testing.T) {
		var logOutput bytes.Buffer
		tracer := newLogTracer(&logOutput)
		logger := tracer.FromContext(ctx).Logger()

		logger.Debug("test message", attrs...)

		const expect = `{"level":"debug","msg":"test message","somestring":"xyz","someint":42,"somebool":false}` + "\n"

		if got := logOutput.String(); got != expect {
			t.Errorf("Expected\n  %sgot\n  %s", expect, got)
		}
	})
}

// newLogTracer returns a NoOpTracerWithLogger configured with a logger that
// writes to the given io.Writer using the JSON encoding.
func newLogTracer(logOutput io.Writer, logOptions ...zap.Option) *NoOpTracerWithLogger {
	zapLogger := newLogger(logOutput, logOptions...)
	return NewNoOpTracerWithLogger(zapLogger.Sugar())
}

// newLogger returns a logger with the same configuration returned by
// zap.NewExample(), except that it writes to the given io.Writer instead of
// os.Stdout.
func newLogger(w io.Writer, options ...zap.Option) *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapcore.AddSync(w), zap.DebugLevel)

	return zap.New(core).WithOptions(options...)
}
