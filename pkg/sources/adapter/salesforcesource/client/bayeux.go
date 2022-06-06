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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/clock"

	"github.com/triggermesh/triggermesh/pkg/sources/adapter/salesforcesource/auth"
)

const connectChannel = "/meta/connect"
const subscribeChannel = "/meta/subscribe"
const handshakeChannel = "/meta/handshake"

// ConnectResponseCallback is the signature for callback functions
// that handle connect responses that contain data.
type ConnectResponseCallback func(msg *ConnectResponse)

// Bayeux client runner
type Bayeux interface {
	Start(ctx context.Context) error
}

// bayeux client according to CometD 3.1.14
// and the subset needed for Salesforce Streaming API.
// See: https://docs.cometd.org/current3/reference/
type bayeux struct {
	auth           auth.Authenticator
	needsHandshake bool
	creds          *auth.Credentials

	client     *http.Client
	clientID   string
	apiVersion string
	url        string

	errCh  chan error
	msgCh  chan *ConnectResponse
	stopCh chan struct{}

	// store replayID per subscription
	subsReplays map[string]*int64

	dispatcher EventDispatcher

	logger *zap.SugaredLogger
	ctx    context.Context
	mutex  sync.RWMutex
}

// NewBayeux creates a Bayeux client for Salesforce Streaming API consumption.
func NewBayeux(apiVersion string, subscriptions []Subscription, authenticator auth.Authenticator, dispatcher EventDispatcher, client *http.Client, logger *zap.SugaredLogger) Bayeux {

	// replayID is stored in a map and will keep track of the latest event received,
	// we copy the configured value for initialization.
	sr := make(map[string]*int64, len(subscriptions))
	for _, s := range subscriptions {
		r := int64(s.ReplayID)
		sr[s.Channel] = &r
	}

	return &bayeux{
		auth:           authenticator,
		needsHandshake: true,

		client:     client,
		apiVersion: apiVersion,

		msgCh:  make(chan *ConnectResponse),
		errCh:  make(chan error),
		stopCh: make(chan struct{}),

		dispatcher:  dispatcher,
		subsReplays: sr,

		logger: logger,
	}
}

// init does handshake and subscriptions.
func (b *bayeux) init() error {
	if err := b.authenticate(); err != nil {
		return err
	}

	if err := b.handshake(); err != nil {
		return err
	}

	for k, v := range b.subsReplays {
		replayID := atomic.LoadInt64(v)
		sbs, err := b.subscribe(k, int(replayID))
		if err != nil {
			return err
		}
		for _, sb := range sbs {
			if !sb.Successful {
				return fmt.Errorf("could not subscribe to %s: %s", sb.Subscription, sb.Error)
			}
		}
	}

	return nil
}

// uses the authenticator to retrieve credentials
func (b *bayeux) authenticate() error {
	creds, err := b.auth.CreateOrRenewCredentials()

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if err != nil {
		// if there was an error with refresh, let's try to
		// use a new access token next time
		b.creds = nil
		return err
	}

	b.creds = creds
	b.url = b.creds.InstanceURL + "/cometd/" + b.apiVersion

	return nil
}

// Start is a blocking function that will loop according to Bayeux,
// autheticating, handshaking, subscripting, and then connecting and
// sending data to the dispatcher until the connection is no longer valid.
// The process is stopped by cancelling the passed context.
func (b *bayeux) Start(ctx context.Context) error {
	// Received context is used for all HTTP calls and
	// at the connect loop to cancel processing.
	b.mutex.Lock()
	b.ctx = ctx
	b.mutex.Unlock()

	bom := wait.NewExponentialBackoffManager(time.Second, time.Second*60, time.Second*100, 2, 0, &clock.RealClock{})

	// Connect loop will run until context is done
	go func() {
		// helper variable for signaling channels
		empty := struct{}{}

		// poll channel receives a signal when it is ok to
		// receive data from Salesforce. We won't send signals
		// to this channel when an error occurs at connection and
		// we need to temporarily retry using backoffs, or when
		// the context is done.
		// All other cases at the processing loop MUST put the signal
		// in the poll channel.
		poll := make(chan struct{}, 1)

		// errc channel receives a signal when a connection error occurs.
		errc := make(chan struct{}, 1)

		// we signal the poll channel at the beginning to start the processing.
		poll <- empty

		for {
			select {
			case <-b.ctx.Done():
				close(b.stopCh)
				return

			case <-errc:
				// return control to the loop to make sure we exit when the
				// context is done. The go routine will send a signal to
				// the poll channel when the backoff timer has ended.
				go func() {
					<-bom.Backoff().C()
					poll <- empty
				}()

			case <-poll:

				// default:
				if b.needsHandshake {
					if err := b.init(); err != nil {
						b.errCh <- err
						// init was not successful, either the handshake or the subscription
						// failed, we need to continue with the next loop iteration which will
						// retry the operation.

						// backing off to avoid locking the Salesforce account.
						errc <- empty
						continue
					}

					// we are good to receive from channels, remove handshake flag
					b.mutex.Lock()
					b.needsHandshake = false
					b.mutex.Unlock()
				}

				crs, err := b.Connect()
				if err != nil {
					b.errCh <- err

					// backing off to avoid locking the Salesforce account.
					errc <- empty
					continue
				}

				for i := range crs {
					// if the message comes from meta channel it is processed immediately and
					// not sent to a processing channel. That way we avoid launching new connects
					//  when some action (handshake) needs to be taken.
					if strings.HasPrefix(crs[i].Channel, "/meta") {
						b.manageMeta(&crs[i])
						continue
					}
					b.msgCh <- &crs[i]
				}

				// process next item.
				poll <- empty
			}
		}
	}()

	// Worker loop will run until the connect loop is stopped.
	for {
		select {
		case msg := <-b.msgCh:
			b.dispatcher.DispatchEvent(b.ctx, msg)
			r, ok := b.subsReplays[msg.Channel]
			if ok {
				atomic.StoreInt64(r, msg.Data.Event.ReplayID)
			}
		case err := <-b.errCh:
			b.dispatcher.DispatchError(err)
		case <-b.stopCh:
			return nil
		}
	}
}

func (b *bayeux) manageMeta(cr *ConnectResponse) {
	if cr.Successful {
		b.logger.Debugf("meta channel (channel: %s client: %s) ok", cr.Channel, cr.ClientID)
		return
	}

	b.logger.Warnf("meta channel (channel: %s client: %s) was not successful: %+v", cr.Channel, cr.ClientID, *cr)

	if cr.Advice.Reconnect == "handshake" {
		b.logger.Debug("marking handshake needed as advised by channel response")
		b.mutex.Lock()
		defer b.mutex.Unlock()
		b.needsHandshake = true
	}
}

// handshake should only be called when a new session is needed, which
// will be when starting the stream process and when auth errors are
// received from any call.
func (b *bayeux) handshake() error {
	payload := `{"channel": "` + handshakeChannel + `", "supportedConnectionTypes": ["long-polling"], "version": "1.0"}`

	// lock to write CookieJar and ClientID
	b.mutex.Lock()
	defer b.mutex.Unlock()

	jOpts := cookiejar.Options{PublicSuffixList: publicsuffix.List}
	jar, err := cookiejar.New(&jOpts)
	if err != nil {
		return fmt.Errorf("could not setup cookiejar: %+w", err)
	}

	b.client.Jar = jar
	res, err := b.doPost(payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	h := []HandshakeResponse{}
	err = json.NewDecoder(res.Body).Decode(&h)
	if err != nil {
		return fmt.Errorf("could not decode handshake response: %+w", err)
	}

	if len(h) == 0 {
		return errors.New("empty handshake response")
	}

	if !h[0].Successful {
		return fmt.Errorf("handshake failed: %s", h[0].Error)
	}

	// store client ID for further connect calls
	b.clientID = h[0].ClientID

	return nil
}

// Connect will start to receive events.
// This is a blocking function
func (b *bayeux) Connect() ([]ConnectResponse, error) {
	payload := `{"channel": "` + connectChannel + `", "connectionType": "long-polling", "clientId": "` + b.clientID + `"}`

	res, err := b.doPost(payload)
	if err != nil {
		return nil, fmt.Errorf("error sending connect request: %+w", err)
	}
	defer res.Body.Close()

	c := []ConnectResponse{}
	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		return nil, fmt.Errorf("could not decode connect response: %+w", err)
	}

	if len(c) == 0 {
		return nil, errors.New("empty connect response")
	}

	return c, nil
}

func (b *bayeux) subscribe(topic string, replayID int) ([]SubscriptionResponse, error) {
	payload := `{"channel": "` + subscribeChannel + `", "subscription": "` + topic + `", "clientId": "` + b.clientID + `","ext":{"replay": {"` + topic + `": ` + strconv.Itoa(replayID) + `}}}`
	res, err := b.doPost(payload)

	if err != nil {
		return nil, fmt.Errorf("error sending subscription request: %+w", err)
	}
	defer res.Body.Close()

	s := []SubscriptionResponse{}
	err = json.NewDecoder(res.Body).Decode(&s)

	if err != nil {
		return nil, fmt.Errorf("could not decode subscription response: %+w", err)
	}

	if len(s) == 0 {
		return nil, errors.New("empty subscription response")
	}

	return s, nil
}

func (b *bayeux) doPost(payload string) (*http.Response, error) {
	req, err := http.NewRequest("POST", b.url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, fmt.Errorf("could not build request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+b.creds.Token)

	req = req.WithContext(b.ctx)
	res, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not execute request: %w", err)
	}

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		b.mutex.Lock()
		b.needsHandshake = true
		b.mutex.Unlock()
	}

	if res.StatusCode >= 300 {
		msg := fmt.Sprintf("received unexpected status code %d", res.StatusCode)
		if resb, err := ioutil.ReadAll(res.Body); err == nil {
			msg += ": " + string(resb)
		}
		return nil, errors.New(msg)
	}

	return res, nil
}
