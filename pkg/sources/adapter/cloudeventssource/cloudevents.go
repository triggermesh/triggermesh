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
Contains code excerpt based on VMware's VEBA's webhook

vCenter Event Broker Appliance
Copyright (c) 2019 VMware, Inc.  All rights reserved

The BSD-2 license (the "License") set forth below applies to all parts of the vCenter Event Broker Appliance project.  You may not use this file except in compliance with the License.

BSD-2 License

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package cloudeventssource

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"

	"github.com/triggermesh/triggermesh/pkg/adapter/fs"
)

type cloudEventsHandler struct {
	basicAuths KeyMountedValues

	cfw      fs.CachedFileWatcher
	ceServer cloudevents.Client
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag
}

// Start implements adapter.Adapter.
func (h *cloudEventsHandler) Start(ctx context.Context) error {
	h.cfw.Start(ctx)
	return h.ceServer.StartReceiver(ctx, h.handle)
}

func (h *cloudEventsHandler) handle(ctx context.Context, e event.Event) protocol.Result {
	err := e.Validate()
	if err != nil {
		h.logger.Errorw("Incoming CloudEvent is not valid", zap.Error(err))
		return protocol.ResultNACK
	}

	result := h.ceClient.Send(ctx, e)
	if !cloudevents.IsACK(result) {
		h.logger.Errorw("Could not send CloudEvent", zap.Error(result))
	}

	return result
}

// code based on VMware's VEBA's webhook:
// https://github.com/vmware-samples/vcenter-event-broker-appliance/blob/e91e4bd8a17dad6ce4fe370c42a15694c03dac88/vmware-event-router/internal/provider/webhook/webhook.go#L167-L189
func (h *cloudEventsHandler) handleAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if ok {
			// reduce brute-force guessing attacks with constant-time comparisons
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))

			for _, kv := range h.basicAuths {
				p, err := h.cfw.GetContent(kv.MountedValueFile)
				if err != nil {
					h.logger.Errorw(
						fmt.Sprintf("Could not retrieve password for user %q", kv.Key),
						zap.Error(err))
					continue
				}

				expectedUsernameHash := sha256.Sum256([]byte(kv.Key))
				expectedPasswordHash := sha256.Sum256(p)

				usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
				passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1

				if usernameMatch && passwordMatch {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
