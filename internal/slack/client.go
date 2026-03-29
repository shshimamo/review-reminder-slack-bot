package slack

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type Client struct {
	api *slack.Client
}

func NewClient(token string) *Client {
	return &Client{
		api: slack.New(token),
	}
}

type PRMessage struct {
	Owner    string
	Repo     string
	Number   int
	URL      string
	Mentions []string // raw mention strings like <@U123>, <!subteam^S123>
}

type ChannelMessage struct {
	Text      string
	Reactions []string
}

var (
	prLinkRegex  = regexp.MustCompile(`https?://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
	mentionRegex = regexp.MustCompile(`<@[^>]+>|<!subteam\^[^>]+>`)
)

// GetYesterdayMessages retrieves messages from the channel posted yesterday.
func (c *Client) GetYesterdayMessages(channelID string) ([]ChannelMessage, error) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
	endOfYesterday := startOfYesterday.AddDate(0, 0, 1)

	oldest := fmt.Sprintf("%d", startOfYesterday.Unix())
	latest := fmt.Sprintf("%d", endOfYesterday.Unix())

	var allMessages []ChannelMessage
	cursor := ""

	for {
		params := &slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Oldest:    oldest,
			Latest:    latest,
			Limit:     200,
			Cursor:    cursor,
		}

		resp, err := c.api.GetConversationHistory(params)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversation history: %w", err)
		}

		for _, msg := range resp.Messages {
			var reactions []string
			for _, r := range msg.Reactions {
				reactions = append(reactions, r.Name)
			}
			allMessages = append(allMessages, ChannelMessage{
				Text:      msg.Text,
				Reactions: reactions,
			})
		}

		if !resp.HasMore {
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}

	return allMessages, nil
}

// ExtractPRMessages extracts GitHub PR links and mentions from messages.
func ExtractPRMessages(messages []ChannelMessage, completeStamp string) []PRMessage {
	var prMessages []PRMessage
	seen := make(map[string]bool)

	for _, msg := range messages {
		// Skip if message has complete stamp
		if hasCompleteStamp(msg.Reactions, completeStamp) {
			continue
		}

		matches := prLinkRegex.FindAllStringSubmatch(msg.Text, -1)
		if len(matches) == 0 {
			continue
		}

		mentions := mentionRegex.FindAllString(msg.Text, -1)

		for _, match := range matches {
			owner := match[1]
			repo := match[2]
			number := match[3]
			url := match[0]

			key := fmt.Sprintf("%s/%s#%s", owner, repo, number)
			if seen[key] {
				continue
			}
			seen[key] = true

			var num int
			fmt.Sscanf(number, "%d", &num)

			prMessages = append(prMessages, PRMessage{
				Owner:    owner,
				Repo:     repo,
				Number:   num,
				URL:      url,
				Mentions: mentions,
			})
		}
	}

	return prMessages
}

func hasCompleteStamp(reactions []string, completeStamp string) bool {
	for _, r := range reactions {
		if r == completeStamp {
			return true
		}
	}
	return false
}

// PostMessage sends a message to the channel.
func (c *Client) PostMessage(channelID, text string) error {
	_, _, err := c.api.PostMessage(channelID, slack.MsgOptionText(text, false))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}

// FormatReminderMessage builds the reminder message text.
func FormatReminderMessage(reminders []Reminder) string {
	if len(reminders) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("*:eyes: レビューリマインド*\n\n")

	for _, r := range reminders {
		b.WriteString(fmt.Sprintf("<%s|%s/%s#%d> - %s\n", r.URL, r.Owner, r.Repo, r.Number, r.Title))
		b.WriteString(fmt.Sprintf("  %s\n", r.StatusText))
		if len(r.Mentions) > 0 {
			b.WriteString(fmt.Sprintf("  %s\n", strings.Join(r.Mentions, " ")))
		}
		b.WriteString("\n")
	}

	return b.String()
}

type Reminder struct {
	Owner      string
	Repo       string
	Number     int
	URL        string
	Title      string
	StatusText string
	Mentions   []string
}
