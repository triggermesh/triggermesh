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

package javascript

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/clbanning/mxj"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"

	"github.com/triggermesh/triggermesh/pkg/targets/adapter/infratarget/vm"
)

type javascriptVM struct {
	pool     *sync.Pool
	timeout  time.Duration
	template string
}

// New creates a new VM that can run a scoped virtual machine
func New(script string, timeout time.Duration, logger *zap.SugaredLogger) vm.InfraVM {
	// escaping to avoid issues with go formatting
	script = strings.ReplaceAll(script, "%", "%%")
	template := script + `

	(function(){
		if ( typeof handle !== "function") {
			throw "script does not implement handle(input) function";
		}
		if ( handle.length != 1 ) {
			throw "handle(input) function accepts exactly one parameter";
		}

		// Avoid issues with escaped chars before JSON parsing
		inParam = '%s';
		var escapeChars = {
			'\b': '\\b',
			'\t': '\\t',
			'\n': '\\n',
			'\f': '\\f',
			'\r': '\\r',
		};

		inParam = inParam.replace(/[\b\t\n\f\r]/g, function(c) {return escapeChars[c]});
		input = JSON.parse(inParam);

		return handle(input)
	})();
	`

	pool := sync.Pool{
		New: func() interface{} {
			o := otto.New()
			o.Interrupt = make(chan func(), 1)
			if err := o.Set("log", vmLogger(logger)); err != nil {
				logger.Errorw("Could not inject logger inside virtual machine", zap.Error(err))
			}
			return o
		},
	}

	return &javascriptVM{
		pool:     &pool,
		timeout:  timeout,
		template: template,
	}
}

// Exec runs the embedded user script using the CloudEvent as a parameter.
func (j *javascriptVM) Exec(event *cloudevents.Event) (*cloudevents.Event, error) {

	if dmt := event.DataMediaType(); dmt == "application/xml" || dmt == "text/xml" {
		xml, err := mxj.NewMapXml(event.Data())
		if err != nil {
			return nil, fmt.Errorf("error parsing XML event data contents: %w", err)
		}
		if err = event.SetData(cloudevents.ApplicationJSON, xml.Old()); err != nil {
			return nil, fmt.Errorf("error setting JSON event data converted from XML: %w", err)
		}
	}

	b, err := event.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error serializing event: %w", err)
	}

	script := fmt.Sprintf(j.template, b)

	jsres, err := j.timeScopedExec(script)
	if err != nil {
		return nil, fmt.Errorf("error running JS script: %v", err)
	}

	res, err := jsres.Export()
	if err != nil {
		return nil, fmt.Errorf("error retrieving result: %v", err)
	}

	// we expect res to be a response cloud event or nil.
	switch t := res.(type) {
	case map[string]interface{}:
		// defaulting missing fields, we do not check CloudEvent type
		// because it is most probably used at filters and defaulting
		// to the incoming one might end up in a loop.
		if _, ok := t["specversion"]; !ok {
			t["specversion"] = event.SpecVersion()
		}
		if _, ok := t["source"]; !ok {
			t["source"] = event.Source()
		}
		if _, ok := t["id"]; !ok {
			t["id"] = event.ID()
		}
		if _, ok := t["datacontenttype"]; !ok {
			t["datacontenttype"] = cloudevents.ApplicationJSON
		}

		b, err := json.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("error serializing exec response: %w", err)
		}

		event := &cloudevents.Event{}
		err = event.UnmarshalJSON(b)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling exec response into event: %w", err)
		}
		return event, nil

	case nil:
		return nil, nil

	default:
		err = fmt.Errorf("unexpected execution result type %T: %v", t, t)
	}

	return nil, err
}

var errTimeout = errors.New("VM execution timed out")

// timeScopedExec runs the script in an scoped time window
func (j *javascriptVM) timeScopedExec(script string) (res *otto.Value, err error) {
	ovm := j.pool.Get().(*otto.Otto)

	ctx, cancel := context.WithTimeout(context.Background(), j.timeout)
	defer cancel()

	ch := make(chan struct{})
	var jres otto.Value
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if r.(error) == errTimeout {
					// we only need to recover to avoid panic from propagating,
					// the actual function result is set by the parent routing.
					return
				}

				// an error ocurred and it is not a timeout
				res = nil
				err = r.(error)
			}
		}()

		jres, err = ovm.Run(script)
		res = &jres
		j.pool.Put(ovm)
		ch <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		ovm.Interrupt <- func() {
			panic(errTimeout)
		}
		return nil, errTimeout
	case <-ch:
		return res, err
	}
}

// vmLogger returns a function that manages log messages
// from the virtual machine instance.
func vmLogger(logger *zap.SugaredLogger) func(string) {
	return func(msg string) {
		logger.Info(msg)
	}
}
