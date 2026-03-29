package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
)

type Client struct {
	client *github.Client
}

func NewClient(token string) *Client {
	return &Client{
		client: github.NewClient(nil).WithAuthToken(token),
	}
}

type ReviewStatus struct {
	Owner     string
	Repo      string
	Number    int
	Title     string
	IsMerged  bool
	IsClosed  bool
	Approvals []string // Approve 済みのレビュアーのログイン名
}

func (c *Client) GetReviewStatus(ctx context.Context, owner, repo string, number int) (*ReviewStatus, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR %s/%s#%d: %w", owner, repo, number, err)
	}

	status := &ReviewStatus{
		Owner:    owner,
		Repo:     repo,
		Number:   number,
		Title:    pr.GetTitle(),
		IsMerged: pr.GetMerged(),
		IsClosed: pr.GetState() == "closed",
	}

	if status.IsMerged || status.IsClosed {
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
