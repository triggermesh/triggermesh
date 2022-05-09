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

// FakeCloudEventsClient is an implementation of a test CloudEvents client.
type FakeCloudEventsClient struct {
	lock sync.Mutex
	sent []cloudevents.Event
	fn   interface{}
}

var _ cloudevents.Client = (*FakeCloudEventsClient)(nil)

// Send is a mock method to send events.
func (c *FakeCloudEventsClient) Send(ctx context.Context, out event.Event) protocol.Result {
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

// Request is a mock method to send events.
func (c *FakeCloudEventsClient) Request(ctx context.Context, out event.Event) (*event.Event, protocol.Result) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// TODO: improve later.
	c.sent = append(c.sent, out)
	return nil, http.NewResult(200, "%w", protocol.ResultACK)
}

// StartReceiver is a mock method to start events receiver.
func (c *FakeCloudEventsClient) StartReceiver(ctx context.Context, fn interface{}) error {
	c.setDispatcher(fn)
	<-ctx.Done()
	return nil
}

func (c *FakeCloudEventsClient) setDispatcher(fn interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.fn = fn
}

// Reset flushes events array.
func (c *FakeCloudEventsClient) Reset() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.sent = make([]cloudevents.Event, 0)
}

// IsReceiverReady returns true if the dispatcher
func (c *FakeCloudEventsClient) IsReceiverReady() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.fn != nil
}

// WaitForReceiverReady will poll the client to check if the receiver dispatcher function is configured.
func (c *FakeCloudEventsClient) WaitForReceiverReady(wait time.Duration) error {
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

// Sent returns the slice of events that were sent with test client.
func (c *FakeCloudEventsClient) Sent() []cloudevents.Event {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := make([]cloudevents.Event, len(c.sent))
	copy(r, c.sent)
	return r
}

// NewTestClient returns a new instance of test client.
func NewTestClient() *FakeCloudEventsClient {
	c := &FakeCloudEventsClient{
		sent: make([]cloudevents.Event, 0),
	}
	return c
}
