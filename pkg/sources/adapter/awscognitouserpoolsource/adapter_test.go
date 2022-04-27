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

package awscognitouserpoolsource

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

type mockedCognitoUserPoolClient struct {
	cognitoidentityprovideriface.CognitoIdentityProviderAPI
	listUsersOutput cognitoidentityprovider.ListUsersOutput
}

func (m mockedCognitoUserPoolClient) ListUsers(in *cognitoidentityprovider.ListUsersInput) (*cognitoidentityprovider.ListUsersOutput, error) {
	return &m.listUsersOutput, nil
}

func (mockedCognitoUserPoolClient) DescribeUserPool(*cognitoidentityprovider.DescribeUserPoolInput) (*cognitoidentityprovider.DescribeUserPoolOutput, error) {
	return &cognitoidentityprovider.DescribeUserPoolOutput{}, nil
}

func TestListUsers(t *testing.T) {
	user := cognitoidentityprovider.UserType{
		UserLastModifiedDate: aws.Time(time.Now()),
	}

	a := &adapter{
		userPoolID: "userpool/fooPool",
		logger:     loggingtesting.TestLogger(t),
		ceClient:   adaptertest.NewTestClient(),
		cgnIdentityClient: mockedCognitoUserPoolClient{
			listUsersOutput: cognitoidentityprovider.ListUsersOutput{
				Users: []*cognitoidentityprovider.UserType{
					&user,
				},
			},
		},
	}

	users, err := a.listUsers()
	assert.NoError(t, err)
	assert.Equal(t, []*cognitoidentityprovider.UserType{&user}, users)
}

func TestFilterByTimestamp(t *testing.T) {
	now := time.Unix(0, 0)

	// existing user
	user1 := cognitoidentityprovider.UserType{
		UserLastModifiedDate: &now,
	}

	// we must get 0 new users
	users, timestamp := filterByTimestamp([]*cognitoidentityprovider.UserType{
		&user1,
	}, now)
	assert.Len(t, users, 0)

	// new user
	user2 := cognitoidentityprovider.UserType{
		UserLastModifiedDate: aws.Time(time.Unix(0, 1)),
	}

	// we must get user2 only
	users, timestamp = filterByTimestamp([]*cognitoidentityprovider.UserType{
		&user1,
		&user2,
	}, timestamp)
	assert.Equal(t, []*cognitoidentityprovider.UserType{&user2}, users)

	// repeat filtering, we must ignore user2
	users, _ = filterByTimestamp([]*cognitoidentityprovider.UserType{
		&user1,
		&user2,
	}, timestamp)
	assert.Len(t, users, 0)
}

func TestSendCognitoEvent(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	user := cognitoidentityprovider.UserType{
		UserLastModifiedDate: aws.Time(time.Now().UTC()),
	}

	a := &adapter{
		userPoolID:        "userpool/fooPool",
		logger:            loggingtesting.TestLogger(t),
		ceClient:          ceClient,
		cgnIdentityClient: mockedCognitoUserPoolClient{},
	}

	ctx := context.Background()

	err := a.sendCognitoEvent(ctx, &user)
	assert.NoError(t, err)

	events := ceClient.Sent()
	assert.Len(t, events, 1)

	var gotUser cognitoidentityprovider.UserType
	err = events[0].DataAs(&gotUser)
	assert.NoError(t, err)
	// make sure that we get what we sent
	assert.Equal(t, user, gotUser)
}

func TestStart(t *testing.T) {
	const testTimeout = 2 * time.Second

	a := &adapter{
		userPoolID:        "userpool/fooPool",
		logger:            loggingtesting.TestLogger(t),
		ceClient:          adaptertest.NewTestClient(),
		cgnIdentityClient: mockedCognitoUserPoolClient{},
	}

	// create a context that will be done after testTimeout
	testCtx, testCancel := context.WithTimeout(context.Background(), testTimeout)
	defer testCancel()

	// context to receive stop signals
	adapCtx, adapCancel := context.WithCancel(context.Background())

	// channel to receive the value returned by Start()
	errCh := make(chan error)

	// run adapter handler in the background
	go func() {
		errCh <- a.Start(adapCtx)
	}()

	// simulate sending a stop signal to the adapter
	adapCancel()

	// if the timeout is done before the handler returned, the test is always failed
	select {
	case <-testCtx.Done():
		t.Errorf("Test timed out after %v", testTimeout)
	case err := <-errCh:
		assert.NoError(t, err, "Receiver returned an error")
	}
}
