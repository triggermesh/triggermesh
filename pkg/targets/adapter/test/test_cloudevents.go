/*
Copyright 2021 TriggerMesh Inc.

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

package test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/cloudevents/sdk-go/v2/protocol/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type TestCloudEventsClient struct {
	lock sync.Mutex
	sent []cloudevents.Event
	fn   interface{}
}

var _ cloudevents.Client = (*TestCloudEventsClient)(nil)

func (c *TestCloudEventsClient) Send(ctx context.Context, out event.Event) protocol.Result {
	c.lock.Lock()
	defer c.lock.Unlock()
	// TODO: improve later.
	c.sent = append(c.sent, out)

	var res protocol.Result
	switch fn := c.fn.(type) {
	case nil:
		return protocol.ResultACK
	case func(event cloudevents.Event) protocol.Result:
		return fn(out)
	}

	return http.NewResult(200, "%w", res)
}

func (c *TestCloudEventsClient) Request(ctx context.Context, out event.Event) (*event.Event, protocol.Result) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// TODO: improve later.
	c.sent = append(c.sent, out)
	return nil, http.NewResult(200, "%w", protocol.ResultACK)
}

func (c *TestCloudEventsClient) StartReceiver(ctx context.Context, fn interface{}) error {
	c.setDispatcher(fn)
	<-ctx.Done()
	return nil
}

func (c *TestCloudEventsClient) setDispatcher(fn interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.fn = fn
}

func (c *TestCloudEventsClient) Reset() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.sent = make([]cloudevents.Event, 0)
}

// IsReceiverReady returns true if the dispatcher
func (c *TestCloudEventsClient) IsReceiverReady() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.fn != nil
}

// WaitForReceiverReady will poll the client to check if the receiver dispatcher function is configured.
func (c *TestCloudEventsClient) WaitForReceiverReady(wait time.Duration) error {
	timeout := time.After(wait)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if c.IsReceiverReady() {
				return nil
			}
		case <-timeout:
			return errors.New("timed out waiting for cloud events client dispatcher")
		}
	}

}

func (c *TestCloudEventsClient) Sent() []cloudevents.Event {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := make([]cloudevents.Event, len(c.sent))
	for i := range c.sent {
		r[i] = c.sent[i]
	}
	return r
}

func NewTestClient() *TestCloudEventsClient {
	c := &TestCloudEventsClient{
		sent: make([]cloudevents.Event, 0),
	}
	return c
}
