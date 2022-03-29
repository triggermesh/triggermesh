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

package slacksource

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	signatureHeader          = "X-Slack-Signature"
	signatureTimestampHeader = "X-Slack-Request-Timestamp"
	expiresSeconds           = int64(300)
)

// timeWrap allows for mocking Now functions at tests.
type timeWrap interface {
	Now() time.Time
}

type standardTime struct{}

func (standardTime) Now() time.Time {
	return time.Now()
}

var _ timeWrap = (*standardTime)(nil)

// verifySigning using signature headers and request body hash.
// see: https://api.slack.com/authentication/verifying-requests-from-slack
func (h *slackEventAPIHandler) verifySigning(header http.Header, body []byte) error {
	signature := sanitizeUserInput(header.Get(signatureHeader))
	if signature == "" {
		return errors.New("empty signature header")
	}

	if signature[:3] != "v0=" {
		return errors.New(`signature header format does not begin with "v0=": ` + signature)
	}

	timestamp := header.Get(signatureTimestampHeader)
	if timestamp == "" {
		return errors.New("empty signature timestamp header")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing header timestamp: %w", err)
	}

	now := int64(h.time.Now().Unix())
	if now-ts > expiresSeconds {
		return errors.New("signing timestamp expired")
	}

	signString := "v0:" + timestamp + ":" + string(body)
	hm := hmac.New(sha256.New, []byte(h.signingSecret))
	if _, err := hm.Write([]byte(signString)); err != nil {
		return fmt.Errorf("error writing signing string into hmac: %w", err)
	}

	hash := hm.Sum(nil)
	challenge := hex.EncodeToString(hash)

	// remove `v=0` from signature
	if challenge != signature[3:] {
		return errors.New("received wrong signature signing hash")
	}

	return nil
}
