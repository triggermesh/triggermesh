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

package awscodecommitsource

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/aws/aws-sdk-go/service/codecommit/codecommitiface"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

type mockedClientForPush struct {
	codecommitiface.CodeCommitAPI
	GetBranchResp codecommit.GetBranchOutput
	GetCommitResp codecommit.GetCommitOutput
	GetBranchErr  error
	GetCommitErr  error
}

type mockedClientForPR struct {
	codecommitiface.CodeCommitAPI
	ListPRsResp codecommit.ListPullRequestsOutput
	GetPRResp   codecommit.GetPullRequestOutput
	ListPRsErr  error
	GetPRErr    error
}

func (m mockedClientForPush) GetBranch(in *codecommit.GetBranchInput) (*codecommit.GetBranchOutput, error) {
	return &m.GetBranchResp, m.GetBranchErr
}

func (m mockedClientForPush) GetCommit(in *codecommit.GetCommitInput) (*codecommit.GetCommitOutput, error) {
	return &m.GetCommitResp, m.GetCommitErr
}

func (m mockedClientForPR) ListPullRequests(in *codecommit.ListPullRequestsInput) (*codecommit.ListPullRequestsOutput, error) {
	return &m.ListPRsResp, m.ListPRsErr
}

func (m mockedClientForPR) GetPullRequest(in *codecommit.GetPullRequestInput) (*codecommit.GetPullRequestOutput, error) {
	return &m.GetPRResp, m.GetPRErr
}

func TestSendPREvent(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	a := &adapter{
		logger:   loggingtesting.TestLogger(t),
		ceClient: ceClient,
	}

	pr := &codecommit.PullRequest{}
	pr.SetPullRequestId("12345")

	ctx := context.Background()

	err := a.sendEvent(ctx, pr)
	assert.NoError(t, err)

	gotEvents := ceClient.Sent()
	assert.Len(t, gotEvents, 1, "Expected 1 event, got %d", len(gotEvents))

	var gotData codecommit.PullRequest
	err = gotEvents[0].DataAs(&gotData)
	assert.NoError(t, err)
	assert.EqualValues(t, *pr, gotData, "Expected event %q, got %q", *pr, gotData)
}

func TestSendPushEvent(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	a := &adapter{
		logger:   loggingtesting.TestLogger(t),
		ceClient: ceClient,
	}

	commit := &codecommit.Commit{}
	commit.SetCommitId("12345")

	ctx := context.Background()

	err := a.sendEvent(ctx, commit)
	assert.NoError(t, err)

	gotEvents := ceClient.Sent()
	assert.Len(t, gotEvents, 1, "Expected 1 event, got %d", len(gotEvents))

	var gotData codecommit.Commit
	err = gotEvents[0].DataAs(&gotData)
	assert.NoError(t, err)
	assert.EqualValues(t, *commit, gotData, "Expected event %q, got %q", *commit, gotData)
}

func TestProcessCommits(t *testing.T) {
	lastCommit = "foo"

	testCases := []struct {
		GetBranchResp codecommit.GetBranchOutput
		GetCommitResp codecommit.GetCommitOutput
		GetBranchErr  error
		GetCommitErr  error
		ErrMsg        *string
	}{
		{
			GetBranchResp: codecommit.GetBranchOutput{},
			GetBranchErr:  errors.New("fake get branch error"),
			GetCommitErr:  nil,
			ErrMsg:        aws.String("failed to get branch info: fake get branch error"),
		},
		{
			GetBranchResp: codecommit.GetBranchOutput{
				Branch: &codecommit.BranchInfo{CommitId: aws.String("123")},
			},
			GetBranchErr: nil,
			GetCommitErr: errors.New("fake get commit error"),
			ErrMsg:       aws.String("failed to get commit info: fake get commit error"),
		},
		{
			GetBranchResp: codecommit.GetBranchOutput{
				Branch: &codecommit.BranchInfo{CommitId: aws.String("123")},
			},
			GetCommitResp: codecommit.GetCommitOutput{
				Commit: &codecommit.Commit{CommitId: aws.String("foo")},
			},
			GetBranchErr: nil,
			GetCommitErr: nil,
			ErrMsg:       nil,
		},
		{
			GetBranchResp: codecommit.GetBranchOutput{
				Branch: &codecommit.BranchInfo{CommitId: aws.String("123")},
			},
			GetCommitResp: codecommit.GetCommitOutput{
				Commit: &codecommit.Commit{CommitId: aws.String("bar")},
			},
			GetBranchErr: nil,
			GetCommitErr: nil,
			ErrMsg:       nil,
		},
	}

	for _, tt := range testCases {
		ccClient := mockedClientForPush{
			GetBranchResp: tt.GetBranchResp,
			GetCommitResp: tt.GetCommitResp,
			GetBranchErr:  tt.GetBranchErr,
			GetCommitErr:  tt.GetCommitErr,
		}

		a := &adapter{
			logger:   loggingtesting.TestLogger(t),
			ccClient: ccClient,
			ceClient: adaptertest.NewTestClient(),
		}

		ctx := context.Background()

		err := a.processCommits(ctx)
		if tt.ErrMsg == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, *tt.ErrMsg)
		}

		lastCommit = "foo"
	}
}

func TestProcessPullRequest(t *testing.T) {
	pullRequestIDs = []*string{aws.String("1")}

	testCases := []struct {
		ListPRsResp codecommit.ListPullRequestsOutput
		GetPRResp   codecommit.GetPullRequestOutput
		ListPRsErr  error
		GetPRErr    error
		ErrMsg      *string
	}{
		{
			ListPRsErr: errors.New("fake list PR error"),
			GetPRErr:   nil,
			ErrMsg:     aws.String("failed to list PRs: fake list PR error"),
		},
		{
			ListPRsResp: codecommit.ListPullRequestsOutput{
				PullRequestIds: []*string{aws.String("1")},
			},
			ListPRsErr: nil,
			GetPRErr:   nil,
			ErrMsg:     nil,
		},
		{
			ListPRsResp: codecommit.ListPullRequestsOutput{
				PullRequestIds: []*string{aws.String("2")},
			},
			GetPRResp: codecommit.GetPullRequestOutput{
				PullRequest: &codecommit.PullRequest{},
			},
			ListPRsErr: nil,
			GetPRErr:   errors.New("fake get PR error"),
			ErrMsg:     aws.String("failed to get PR info: fake get PR error"),
		},
		{
			ListPRsResp: codecommit.ListPullRequestsOutput{
				PullRequestIds: []*string{aws.String("2")},
			},
			GetPRResp: codecommit.GetPullRequestOutput{
				PullRequest: &codecommit.PullRequest{},
			},
			ListPRsErr: nil,
			GetPRErr:   nil,
			ErrMsg:     nil,
		},
		{
			ListPRsResp: codecommit.ListPullRequestsOutput{
				PullRequestIds: []*string{aws.String("2")},
			},
			GetPRResp: codecommit.GetPullRequestOutput{
				PullRequest: &codecommit.PullRequest{},
			},
			ListPRsErr: nil,
			GetPRErr:   nil,
			ErrMsg:     nil,
		},
	}

	for _, tt := range testCases {
		ccClient := mockedClientForPR{
			ListPRsResp: tt.ListPRsResp,
			GetPRResp:   tt.GetPRResp,
			ListPRsErr:  tt.ListPRsErr,
			GetPRErr:    tt.GetPRErr,
		}
		ceClient := adaptertest.NewTestClient()

		a := &adapter{
			logger:   loggingtesting.TestLogger(t),
			ccClient: ccClient,
			ceClient: ceClient,
		}

		_, err := a.preparePullRequests()
		if tt.ErrMsg == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, *tt.ErrMsg)
		}

		pullRequestIDs = []*string{aws.String("1")}
	}
}

func TestRemoveOldPRs(t *testing.T) {
	oldPRs := []*codecommit.PullRequest{
		{PullRequestId: aws.String("1"), PullRequestStatus: aws.String("CREATED")},
		{PullRequestId: aws.String("2"), PullRequestStatus: aws.String("CREATED")},
	}
	newPRs := []*codecommit.PullRequest{
		{PullRequestId: aws.String("1"), PullRequestStatus: aws.String("CREATED")},
		{PullRequestId: aws.String("2"), PullRequestStatus: aws.String("CLOSED")},
		{PullRequestId: aws.String("3"), PullRequestStatus: aws.String("CREATED")},
	}

	expectedPRs := []*codecommit.PullRequest{
		{PullRequestId: aws.String("2"), PullRequestStatus: aws.String("CLOSED")},
		{PullRequestId: aws.String("3"), PullRequestStatus: aws.String("CREATED")},
	}

	prs := removeOldPRs(oldPRs, newPRs)
	t.Log(prs)
	assert.Equal(t, 2, len(prs))
	assert.Equal(t, expectedPRs, prs)
}
