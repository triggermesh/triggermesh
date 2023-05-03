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
	"fmt"
	"strings"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/aws/aws-sdk-go/service/codecommit/codecommitiface"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

var (
	//syncTime       = 10
	lastCommit     string
	pullRequestIDs []*string //nolint:unused
)

const (
	pushEventType = "push"
	prEventType   = "pull_request"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	ARN           string `envconfig:"ARN" required:"true"`
	Branch        string `envconfig:"BRANCH" required:"true"`
	GitEventTypes string `envconfig:"EVENT_TYPES" required:"true"`

	// Assume this IAM Role when access keys provided.
	AssumeIamRole string `envconfig:"AWS_ASSUME_ROLE_ARN"`

	// The environment variables below aren't read from the envConfig struct
	// by the AWS SDK, but rather directly using os.Getenv().
	// They are nevertheless listed here for documentation purposes.
	_ string `envconfig:"AWS_ACCESS_KEY_ID"`
	_ string `envconfig:"AWS_SECRET_ACCESS_KEY"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger
	mt     *pkgadapter.MetricTag

	ccClient codecommitiface.CodeCommitAPI
	ceClient cloudevents.Client

	arn       arn.ARN
	branch    string
	gitEvents string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSCodeCommitSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	arn := common.MustParseARN(env.ARN)

	sess := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(arn.Region).
		WithMaxRetries(5),
	))

	config := &aws.Config{}
	if env.AssumeIamRole != "" {
		config.Credentials = stscreds.NewCredentials(sess, env.AssumeIamRole)
	}

	return &adapter{
		logger: logger,
		mt:     mt,

		ccClient: codecommit.New(sess),
		ceClient: ceClient,

		arn:       arn,
		branch:    env.Branch,
		gitEvents: env.GitEventTypes,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	if err := peekRepo(ctx, a.ccClient, a.arn.Resource); err != nil {
		return fmt.Errorf("unable to access repository %q: %w", a.arn, err)
	}

	health.MarkReady()

	if strings.Contains(a.gitEvents, pushEventType) {
		a.logger.Info("AWS CodeCommit Push events enabled")

		branchInfo, err := a.ccClient.GetBranch(&codecommit.GetBranchInput{
			RepositoryName: &a.arn.Resource,
			BranchName:     &a.branch,
		})
		if err != nil {
			a.logger.Fatalw("Failed to retrieve branch info", zap.Error(err))
		}

		lastCommit = *branchInfo.Branch.CommitId
	}

	if strings.Contains(a.gitEvents, prEventType) {
		a.logger.Info("AWS CodeCommit Pull Request events enabled")

		// get pull request IDs
		pullRequestsOutput, err := a.ccClient.ListPullRequests(&codecommit.ListPullRequestsInput{
			RepositoryName: &a.arn.Resource,
		})
		if err != nil {
			a.logger.Fatalw("Failed to retrieve list of pull requests", zap.Error(err))
		}

		pullRequestIDs = pullRequestsOutput.PullRequestIds
	}

	if !strings.Contains(a.gitEvents, pushEventType) && !strings.Contains(a.gitEvents, prEventType) {
		a.logger.Fatalf("Failed to identify event types in %q. Valid values: (push,pull_request)", a.gitEvents)
	}

	processedPullRequests, err := a.preparePullRequests()
	if err != nil {
		a.logger.Errorw("Failed to process pull requests", zap.Error(err))
	}

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	backoff := common.NewBackoff()

	err = backoff.Run(ctx.Done(), func(ctx context.Context) (bool, error) {
		resetBackoff := false

		if strings.Contains(a.gitEvents, pushEventType) {
			err := a.processCommits(ctx)
			if err != nil {
				a.logger.Errorw("Failed to process commits", zap.Error(err))
				return resetBackoff, nil
			}
		}

		if strings.Contains(a.gitEvents, prEventType) {
			pullRequests, err := a.preparePullRequests()
			if err != nil {
				a.logger.Errorw("Failed to process pull requests", zap.Error(err))
				return resetBackoff, nil
			}

			pullRequests = removeOldPRs(processedPullRequests, pullRequests)

			for _, pr := range pullRequests {
				resetBackoff = true
				err = a.sendEvent(ctx, pr)
				if err != nil {
					a.logger.Errorw("Failed to send PR event", zap.Error(err))
					return resetBackoff, nil
				}
				processedPullRequests = append(processedPullRequests, pr)
			}
		}
		return resetBackoff, nil
	})

	return err
}

func (a *adapter) processCommits(ctx context.Context) error {
	branchInfo, err := a.ccClient.GetBranch(&codecommit.GetBranchInput{
		BranchName:     &a.branch,
		RepositoryName: &a.arn.Resource,
	})
	if err != nil {
		return fmt.Errorf("failed to get branch info: %w", err)
	}

	commitOutput, err := a.ccClient.GetCommit(&codecommit.GetCommitInput{
		CommitId:       branchInfo.Branch.CommitId,
		RepositoryName: &a.arn.Resource,
	})
	if err != nil {
		return fmt.Errorf("failed to get commit info: %w", err)
	}

	if *commitOutput.Commit.CommitId == lastCommit {
		return nil
	}

	lastCommit = *commitOutput.Commit.CommitId

	err = a.sendEvent(ctx, commitOutput.Commit)
	if err != nil {
		return fmt.Errorf("failed to send push event: %w", err)
	}

	return nil
}

func (a *adapter) preparePullRequests() ([]*codecommit.PullRequest, error) {
	pullRequests := []*codecommit.PullRequest{}

	input := codecommit.ListPullRequestsInput{
		RepositoryName: &a.arn.Resource,
	}

	// Get pull request IDs
	pullRequestsOutput, err := a.ccClient.ListPullRequests(&input)
	if err != nil {
		return pullRequests, fmt.Errorf("failed to list PRs: %w", err)
	}

	for _, id := range pullRequestsOutput.PullRequestIds {
		pri := codecommit.GetPullRequestInput{PullRequestId: id}

		prInfo, err := a.ccClient.GetPullRequest(&pri)
		if err != nil {
			return pullRequests, fmt.Errorf("failed to get PR info: %w", err)
		}

		pullRequests = append(pullRequests, prInfo.PullRequest)
	}

	return pullRequests, nil
}

// sendEvent sends an event containing data about a git commit or PR
func (a *adapter) sendEvent(ctx context.Context, codeCommitEvent interface{}) error {
	a.logger.Debug("Sending CodeCommit event")

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetSubject(a.branch)
	event.SetSource(a.arn.String())

	switch codeCommitEvent.(type) {
	case *codecommit.PullRequest:
		event.SetType(v1alpha1.AWSEventType(a.arn.Service, prEventType))
	case *codecommit.Commit:
		event.SetType(v1alpha1.AWSEventType(a.arn.Service, pushEventType))
	default:
		return fmt.Errorf("unknown CodeCommit event")
	}

	err := event.SetData(cloudevents.ApplicationJSON, codeCommitEvent)
	if err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}

func removeOldPRs(oldPrs, newPrs []*codecommit.PullRequest) []*codecommit.PullRequest {
	dct := make(map[string]*codecommit.PullRequest)
	for _, oldPR := range oldPrs {
		dct[*oldPR.PullRequestId] = oldPR
	}

	res := make([]*codecommit.PullRequest, 0)

	for _, newPR := range newPrs {
		if v, exist := dct[*newPR.PullRequestId]; !exist {
			res = append(res, newPR)
			continue
		} else {
			if *newPR.PullRequestStatus == *v.PullRequestStatus {
				continue
			}
			res = append(res, newPR)
		}
	}
	return res
}

// peekRepo verifies that the provided repository exists.
func peekRepo(ctx context.Context, cli codecommitiface.CodeCommitAPI, repoName string) error {
	_, err := cli.GetRepositoryWithContext(ctx, &codecommit.GetRepositoryInput{
		RepositoryName: &repoName,
	})
	return err
}
