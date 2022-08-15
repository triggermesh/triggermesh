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

/*
Package cloudevents provides a CloudEvents library focused on target component
requirements and on how responses should be composed.

# Basics

The package provides a Replier object that should be instantiated as a singleton,
and a set of pubic functions that are further divided into Knative managed and
Custom managed responses.

Knative managed responses contain no payload, they just log and report ACK or NACK
to the interacting channel.

Custom managed responses on the other hand provide information through payload. In the
case of successful responses the payload is provided by the target, but when an error
occurs, a custom managed response includes an EventError payload.

	type EventError struct {
		Code        string		// short string to identify error nature
		Description string		// description from the error object
		Fields interface{}	// additional information
	}

# Replier

The Replier constructor can be customized by passing an array of ReplierOption objects,
being that customization applied to all responses. Customization choices are:

	responseType			how to compose the outgoing response event type.
	responseContentType		which content-type header to choose.
	responseSource			how to compose the outgoing event source.

	responseErrorType		in case of error, same as responseType.
	responseErrorContentType	in case of error, same as responseContentType.

	payloadPolicy			if payload replies should be sent or not.

Knative Managed Functions

	Ack() (*cloudevents.Event, cloudevents.Result)

Is the function to use when we want Knative to acknoledge that the message was delivered
and no reponse payload is being returned.

	ErrorKnativeManaged(event *cloudevents.Event, err error) (*cloudevents.Event, cloudevents.Result)

Sends a non properly delivered message back to the channel, which will decide if it should be
retried, sent to the DLQ or forgotten. A summary of the incoming event that led to the error is added
to the error message returned.

Custom Managed Functions

	Ok(in *cloudevents.Event, payload interface{}, opts ...EventResponseOption) (*cloudevents.Event, cloudevents.Result)

Replies acknowledging the delivery of the message and providing a payload for further use. The incoming
event is passed to help composing header fields for the response. EventResponseOptions will be sequentially
processed and can be used to modify the response before sending out.

Ok responses will have a "category" header set to "success".

	ErrorWithFields(in *cloudevents.Event, code string, reportedErr error, details interface{}, opts ...EventResponseOption) (*cloudevents.Event, cloudevents.Result)

Replies acknowledging the delivery of the message, which means Knative will consider the message as succeeded and wont
retry nor send to the DLQ. The code parameter values are suggested at the package but can be provided by the target, being
recommended that cardinality is kept low to make log analysis easy. If response options for type and content-type are not
explicitly set the ones for the Ok response will be used.

Error responses will have a "category" header set to "error".
*/
package cloudevents

/*
Usage:

	// Create replier that will add bridge header to custom responses and
	// will only reply payloads when an error occurs.
	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(bridgeIdentifier),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicyErrors))

...

	// Create replier that will use a different static type response
	// depending on the incoming type.
	mt = map[string]string{
		"search.snowball.io": "list.snowball.io",
		"update.snowball.io": "product.snowball.io",
	}
	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithMappedResponseType(mt))

...

	// Returning a custom success response that uses the title variable as subject
	replier.Ok(inEvent, bJson, targetce.ResponseWithSubject(title))

...

	// Returning a custom error from the dispatcher, not being
	// able to parse the response from a third party service.
	return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
*/
