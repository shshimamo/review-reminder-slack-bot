package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v69/github"
)

type Client struct {
	client *github.Client
}

// NewClientWithToken は PAT を使って GitHub クライアントを作成する。
func NewClientWithToken(token string) *Client {
	return &Client{
		client: github.NewClient(nil).WithAuthToken(token),
	}
}

// NewClientWithApp は GitHub App を使って GitHub クライアントを作成する。
func NewClientWithApp(appID, installationID int64, privateKey string) (*Client, error) {
	transport, err := ghinstallation.New(http.DefaultTransport, appID, installationID, []byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub App transport: %w", err)
	}
	return &Client{
		client: github.NewClient(&http.Client{Transport: transport}),
	}, nil
}

type ReviewStatus struct {
	Owner            string
	Repo             string
	Number           int
	Title            string
	IsMerged         bool
	IsClosed         bool
	HasPendingReview bool     // レビュー待ちの人/チームがいるか
	Approvals        []string // Approve 済みのレビュアーのログイン名
}

func (c *Client) GetReviewStatus(ctx context.Context, owner, repo string, number int) (*ReviewStatus, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR %s/%s#%d: %w", owner, repo, number, err)
	}

	status := &ReviewStatus{
		Owner:            owner,
		Repo:             repo,
		Number:           number,
		Title:            pr.GetTitle(),
		IsMerged:         pr.GetMerged(),
		IsClosed:         pr.GetState() == "closed",
		HasPendingReview: len(pr.RequestedReviewers) > 0 || len(pr.RequestedTeams) > 0,
	}

	if status.IsMerged || status.IsClosed || !status.HasPendingReview {
		return status, nil
	}

	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, number, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviews for %s/%s#%d: %w", owner, repo, number, err)
	}

	// レビュアーごとの最新のレビュー状態を追跡
	latestReview := make(map[string]string)
	for _, r := range reviews {
		login := r.GetUser().GetLogin()
		state := r.GetState()
		latestReview[login] = state
	}

	for login, state := range latestReview {
		if state == "APPROVED" {
			status.Approvals = append(status.Approvals, login)
		}
	}

	return status, nil
}
