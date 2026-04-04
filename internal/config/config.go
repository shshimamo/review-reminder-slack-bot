package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	SlackBotToken           string
	SlackChannel            string
	GitHubToken             string // PAT 方式
	GitHubAppID             int64  // GitHub App 方式
	GitHubAppPrivateKey     string
	GitHubAppInstallationID int64
	CompleteStamp           string
	RequiredApprovalsNumber int
	DaysAgo                 int
}

func (c *Config) UseGitHubApp() bool {
	return c.GitHubAppID != 0
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

	daysAgo := 14
	if v := os.Getenv("DAYS_AGO"); v != "" {
		daysAgo, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("DAYS_AGO must be a number: %w", err)
		}
	}

	cfg := &Config{
		SlackBotToken:           slackBotToken,
		SlackChannel:            slackChannel,
		CompleteStamp:           completeStamp,
		RequiredApprovalsNumber: requiredApprovals,
		DaysAgo:                 daysAgo,
	}

	// GitHub App 方式
	if appIDStr := os.Getenv("GITHUB_APP_ID"); appIDStr != "" {
		cfg.GitHubAppID, err = strconv.ParseInt(appIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("GITHUB_APP_ID must be a number: %w", err)
		}

		cfg.GitHubAppPrivateKey = os.Getenv("GITHUB_APP_PRIVATE_KEY")
		if cfg.GitHubAppPrivateKey == "" {
			return nil, fmt.Errorf("GITHUB_APP_PRIVATE_KEY is required when GITHUB_APP_ID is set")
		}

		installIDStr := os.Getenv("GITHUB_APP_INSTALLATION_ID")
		if installIDStr == "" {
			return nil, fmt.Errorf("GITHUB_APP_INSTALLATION_ID is required when GITHUB_APP_ID is set")
		}
		cfg.GitHubAppInstallationID, err = strconv.ParseInt(installIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("GITHUB_APP_INSTALLATION_ID must be a number: %w", err)
		}

		return cfg, nil
	}

	// PAT 方式
	cfg.GitHubToken = os.Getenv("GITHUB_TOKEN")
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN or GITHUB_APP_ID is required")
	}

	return cfg, nil
}
