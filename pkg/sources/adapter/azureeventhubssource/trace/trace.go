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

package trace

import (
	"context"

	"github.com/devigned/tab"
	"go.uber.org/zap"
)

// NewNoOpTracerWithLogger returns a NoOpTracerWithLogger initialized with the
// given logger.
func NewNoOpTracerWithLogger(l *zap.SugaredLogger) *NoOpTracerWithLogger {
	return &NoOpTracerWithLogger{
		Logger: l.Desugar(),
	}
}

// NoOpTracerWithLogger is a tab.Tracer implementation that doesn't support any
// tracing capabilities, but provides access to a Logger through a Spanner.
type NoOpTracerWithLogger struct {
	Logger *zap.Logger
}

// noOpLogSpanner is a tab.Spanner implementation that does nothing, besides
// providing access to a Logger.
type noOpLogSpanner struct {
	logger *zap.Logger
}

// zapLogger provides access to a logging facilities backed by a zap.Logger.
type zapLogger struct {
	logger *zap.Logger
}

// Verify implementation of interfaces.
var _ tab.Tracer = (*NoOpTracerWithLogger)(nil)
var _ tab.Spanner = (*noOpLogSpanner)(nil)
var _ tab.Logger = (*zapLogger)(nil)

// StartSpan implements tab.Tracer.
func (t *NoOpTracerWithLogger) StartSpan(ctx context.Context, operationName string, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, &noOpLogSpanner{logger: t.Logger}
}

// StartSpanWithRemoteParent implements tab.Tracer.
func (t *NoOpTracerWithLogger) StartSpanWithRemoteParent(ctx context.Context, operationName string, carrier tab.Carrier, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, &noOpLogSpanner{logger: t.Logger}
}

// FromContext implements tab.Tracer.
func (t *NoOpTracerWithLogger) FromContext(context.Context) tab.Spanner {
	return &noOpLogSpanner{logger: t.Logger}
}

// NewContext implements tab.Tracer.
func (t *NoOpTracerWithLogger) NewContext(parent context.Context, _ tab.Spanner) context.Context {
	return parent
}

// AddAttributes implements tab.Spanner.
func (s *noOpLogSpanner) AddAttributes(...tab.Attribute) {}

// End implements tab.Spanner.
func (s *noOpLogSpanner) End() {}

// Logger implements tab.Spanner.
func (s *noOpLogSpanner) Logger() tab.Logger {
	return &zapLogger{logger: s.logger}
}

// Inject implements tab.Spanner.
func (s *noOpLogSpanner) Inject(tab.Carrier) error {
	return nil
}

// InternalSpan implements tab.Spanner.
func (s *noOpLogSpanner) InternalSpan() interface{} {
	return nil
}

// Info implements tab.Logger.
func (l *zapLogger) Info(msg string, attributes ...tab.Attribute) {
	l.logger.Info(msg, attributesToLogFields(attributes)...)
}

// Error implements tab.Logger.
func (l *zapLogger) Error(err error, attributes ...tab.Attribute) {
	l.logger.Error(err.Error(), attributesToLogFields(attributes)...)
}

// Fatal implements tab.Logger.
func (l *zapLogger) Fatal(msg string, attributes ...tab.Attribute) {
	l.logger.Fatal(msg, attributesToLogFields(attributes)...)
}

// Debug implements tab.Logger.
func (l *zapLogger) Debug(msg string, attributes ...tab.Attribute) {
	l.logger.Debug(msg, attributesToLogFields(attributes)...)
}

// attributesToLogFields converts tab.Attributes into a list of fields that can
// be used by a zap.Logger.
func attributesToLogFields(attributes []tab.Attribute) []zap.Field {
	zapFields := make([]zap.Field, len(attributes))
	for i, a := range attributes {
		zapFields[i] = zap.Any(a.Key, a.Value)
	}

	return zapFields
}
