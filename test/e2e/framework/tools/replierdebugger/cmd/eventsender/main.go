// Copyright [2021] [Jeff Naef]

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// send event to sink

// recieve response from the sink

// send response to the K_DBUG_SINK

package main

import (
	"context"
	"fmt"
	"log"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
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

	// Create an Event.
	event := cloudevents.NewEvent()
	event.SetSource("example/uri")
	event.SetType("example.type")
	event.SetData(cloudevents.ApplicationJSON, map[string]string{"hello": "world"})

	// Set a target.
	ctx := cloudevents.ContextWithTarget(context.Background(), env.Sink)

	// Send that Event.

	e, _ := c.Request(ctx, event)
	fmt.Printf("%+v", e)

	// Change the target.
	ctx = cloudevents.ContextWithTarget(context.Background(), env.DebugSink)

	// Send event to debug sink
	if result := c.Send(ctx, *e); cloudevents.IsUndelivered(result) {
		log.Fatalf("failed to send, %v", result)
	}

}
