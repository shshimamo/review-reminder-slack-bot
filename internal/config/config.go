package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	SlackBotToken           string
	SlackChannel            string
	GitHubToken             string
	CompleteStamp           string
	RequiredApprovalsNumber int
}

func Load() (*Config, error) {
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackBotToken == "" {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN is required")
	}

	slackChannel := os.Getenv("SLACK_CHANNEL")
	if slackChannel == "" {
		return nil, fmt.Errorf("SLACK_CHANNEL is required")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
	}

	completeStamp := os.Getenv("COMPLETE_STAMP")
	if completeStamp == "" {
		return nil, fmt.Errorf("COMPLETE_STAMP is required")
	}

	requiredApprovalsStr := os.Getenv("REQUIRED_APPROVALS_NUMBER")
	if requiredApprovalsStr == "" {
		return nil, fmt.Errorf("REQUIRED_APPROVALS_NUMBER is required")
	}
	requiredApprovals, err := strconv.Atoi(requiredApprovalsStr)
	if err != nil {
		return nil, fmt.Errorf("REQUIRED_APPROVALS_NUMBER must be a number: %w", err)
	}

	return &Config{
		SlackBotToken:           slackBotToken,
		SlackChannel:            slackChannel,
		GitHubToken:             githubToken,
		CompleteStamp:           completeStamp,
		RequiredApprovalsNumber: requiredApprovals,
	}, nil
}
