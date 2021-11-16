package twilio

import (
	"context"
	"net/url"

	"github.com/kevinburke/go-types"
)

const servicesPathPart = "Services"
const verificationsPathPart = "Verifications"
const verificationCheckPart = "VerificationCheck"

type VerifyPhoneNumberService struct {
	client *Client
}

type VerifyPhoneNumber struct {
	Sid         string           `json:"sid"`
	ServiceSid  string           `json:"service_sid"`
	AccountSid  string           `json:"account_sid"`
	To          PhoneNumber      `json:"to"`
	Channel     string           `json:"channel"`
	Status      string           `json:"status"`
	Valid       bool             `json:"valid"`
	Lookup      PhoneLookup      `json:"lookup"`
	Amount      types.NullString `json:"amount"`
	Payee       types.NullString `json:"payee"`
	DateCreated TwilioTime       `json:"date_created"`
	DateUpdated TwilioTime       `json:"date_updated"`
	URL         string           `json:"url"`
}

type CheckPhoneNumber struct {
	Sid         string           `json:"sid"`
	ServiceSid  string           `json:"service_sid"`
	AccountSid  string           `json:"account_sid"`
	To          string           `json:"to"`
	Channel     string           `json:"channel"`
	Status      string           `json:"status"`
	Valid       bool             `json:"valid"`
	Amount      types.NullString `json:"amount"`
	Payee       types.NullString `json:"payee"`
	DateCreated TwilioTime       `json:"date_created"`
	DateUpdated TwilioTime       `json:"date_updated"`
}

// Create calls the Verify API to start a new verification.
// https://www.twilio.com/docs/verify/api-beta/verification-beta#start-new-verification
func (v *VerifyPhoneNumberService) Create(ctx context.Context, verifyServiceID string, data url.Values) (*VerifyPhoneNumber, error) {
	verify := new(VerifyPhoneNumber)
	err := v.client.CreateResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+verificationsPathPart, data, verify)
	return verify, err
}

// Get calls the Verify API to retrieve information about a verification.
// https://www.twilio.com/docs/verify/api-beta/verification-beta#fetch-a-verification-1
func (v *VerifyPhoneNumberService) Get(ctx context.Context, verifyServiceID string, sid string) (*VerifyPhoneNumber, error) {
	verify := new(VerifyPhoneNumber)
	err := v.client.GetResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+verificationsPathPart, sid, verify)
	return verify, err
}

// Check calls the Verify API to check if a user-provided token is correct.
// https://www.twilio.com/docs/verify/api-beta/verification-check-beta#check-a-verification-1
func (v *VerifyPhoneNumberService) Check(ctx context.Context, verifyServiceID string, data url.Values) (*CheckPhoneNumber, error) {
	check := new(CheckPhoneNumber)
	err := v.client.CreateResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+verificationCheckPart, data, check)
	return check, err
}
