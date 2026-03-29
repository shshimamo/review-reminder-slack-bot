package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/shshimamo/review-reminder/internal/config"
	gh "github.com/shshimamo/review-reminder/internal/github"
	sl "github.com/shshimamo/review-reminder/internal/slack"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	slackClient := sl.NewClient(cfg.SlackBotToken)
	githubClient := gh.NewClient(cfg.GitHubToken)

	messages, err := slackClient.GetYesterdayMessages(cfg.SlackChannel)
	if err != nil {
		log.Fatalf("Failed to get messages: %v", err)
	}

	prMessages := sl.ExtractPRMessages(messages, cfg.CompleteStamp)
	if len(prMessages) == 0 {
		log.Println("No PR links found in yesterday's messages")
		return
	}

	ctx := context.Background()
	var reminders []sl.Reminder

	for _, prMsg := range prMessages {
		status, err := githubClient.GetReviewStatus(ctx, prMsg.Owner, prMsg.Repo, prMsg.Number)
		if err != nil {
			log.Printf("Warning: failed to get review status for %s: %v", prMsg.URL, err)
			continue
		}

		if status.IsMerged {
			continue
		}

		approvalCount := len(status.Approvals)
		if approvalCount >= cfg.RequiredApprovalsNumber {
			continue
		}

		var statusText string
		switch {
		case approvalCount == 0:
			statusText = "レビュー未着手"
		default:
			statusText = fmt.Sprintf("%s がレビュー済み / あと%d名必要",
				strings.Join(status.Approvals, ", "),
				cfg.RequiredApprovalsNumber-approvalCount,
			)
		}

		reminders = append(reminders, sl.Reminder{
			Owner:      prMsg.Owner,
			Repo:       prMsg.Repo,
			Number:     prMsg.Number,
			URL:        prMsg.URL,
			Title:      status.Title,
			StatusText: statusText,
			Mentions:   prMsg.Mentions,
		})
	}

	if len(reminders) == 0 {
		log.Println("All PRs are merged or fully reviewed")
		return
	}

	message := sl.FormatReminderMessage(reminders)
	if err := slackClient.PostMessage(cfg.SlackChannel, message); err != nil {
		log.Fatalf("Failed to post reminder: %v", err)
	}

	log.Printf("Posted reminder for %d PR(s)", len(reminders))
}
