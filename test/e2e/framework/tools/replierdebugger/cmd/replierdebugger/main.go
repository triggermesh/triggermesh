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

package main

import (
	"context"
	"fmt"
	"log"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
	EventPayload     map[string]string `envconfig:"EVENT_PAYLOAD" default:"{hello:world}"`
	EventContentType string            `envconfig:"EVENT_CONTENT_TYPE" default:"application/json"`
	EventID          string            `envconfig:"EVENT_ID" default:"12345"`
	EventType        string            `envconfig:"EVENT_TYPE" default:"example.type"`
	EventSource      string            `envconfig:"EVENT_SOURCE" default:"example/uri"`

	Sink      string `envconfig:"K_SINK" required:"true"`
	DebugSink string `envconfig:"K_DEBUG_SINK" required:"true"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("failed to process env var: %s", err)
	}

	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	event := cloudevents.NewEvent()
	event.SetSource(env.EventSource)
	event.SetType(env.EventType)
	event.SetData(env.EventContentType, env.EventPayload)

	ctx := cloudevents.ContextWithTarget(context.Background(), env.Sink)

	e, _ := c.Request(ctx, event)
	fmt.Printf("%+v", e)

	ctx = cloudevents.ContextWithTarget(context.Background(), env.DebugSink)

	if result := c.Send(ctx, *e); cloudevents.IsUndelivered(result) {
		log.Fatalf("failed to send, %v", result)
	}
	// The default client is HTTP.
	ceh, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}
	log.Fatal(ceh.StartReceiver(context.Background(), ceHandler))

}

func ceHandler(e cloudevents.Event) {
	fmt.Printf("%+v", e)
}
