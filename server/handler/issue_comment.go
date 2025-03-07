// Copyright 2018 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"
	"encoding/json"

	"github.com/google/go-github/v56/github"
	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
)

type IssueComment struct {
	Base
}

func (h *IssueComment) Handles() []string {
	return []string{"issue_comment"}
}

func (h *IssueComment) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.IssueCommentEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse issue comment payload")
	}

	repo := event.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	number := event.GetIssue().GetNumber()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, number)

	logger.Debug().Msgf("Received issue_comment %s event", event.GetAction())

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	pr, _, err := client.PullRequests.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number)
	if err != nil {
		return errors.Wrapf(err, "failed to get pull request %s/%s#%d", owner, repoName, number)
	}
	pullCtx := pull.NewGithubContext(client, pr)

	config, err := h.FetchConfigForPR(ctx, client, pr)
	if err != nil {
		return err
	}
	if err := h.ProcessPullRequest(ctx, pullCtx, client, config, pr); err != nil {
		logger.Error().Err(errors.WithStack(err)).Msg("Error processing pull request")
	}

	return nil
}

// type assertion
var _ githubapp.EventHandler = &IssueComment{}
