package slack

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
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
	PostedAt time.Time
}

type ChannelMessage struct {
	Text      string
	Reactions []string
	Timestamp time.Time
}

var (
	prLinkRegex  = regexp.MustCompile(`https?://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
	mentionRegex = regexp.MustCompile(`<@[^>]+>|<!subteam\^[^>]+>`)
)

// GetMessages retrieves messages from the channel posted within the last daysAgo days.
func (c *Client) GetMessages(channelID string, daysAgo int) ([]ChannelMessage, error) {
	now := time.Now()
	startDay := now.AddDate(0, 0, -daysAgo)
	startOfRange := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, now.Location())

	oldest := fmt.Sprintf("%d", startOfRange.Unix())

	var allMessages []ChannelMessage
	cursor := ""

	for {
		params := &slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Oldest:    oldest,
			Limit:     200,
			Cursor:    cursor,
		}

		resp, err := c.api.GetConversationHistory(params)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversation history: %w", err)
		}

		for _, msg := range resp.Messages {
			if msg.BotID != "" {
				continue
			}
			var reactions []string
			for _, r := range msg.Reactions {
				reactions = append(reactions, r.Name)
			}
			ts := parseSlackTimestamp(msg.Timestamp, now.Location())
			allMessages = append(allMessages, ChannelMessage{
				Text:      msg.Text,
				Reactions: reactions,
				Timestamp: ts,
			})
		}

		if !resp.HasMore {
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}

	return allMessages, nil
}

func parseSlackTimestamp(ts string, loc *time.Location) time.Time {
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return time.Time{}
	}
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0).In(loc)
}

// ExtractPRMessages extracts GitHub PR links and mentions from messages.
func ExtractPRMessages(messages []ChannelMessage, completeStamp string) []PRMessage {
	// Sort oldest first so the earliest post is adopted for duplicate PRs
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	var prMessages []PRMessage
	seen := make(map[string]bool)

	for _, msg := range messages {
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
				PostedAt: msg.Timestamp,
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

// FormatReminderMessage builds the reminder message text grouped by posting date.
func FormatReminderMessage(reminders []Reminder) string {
	if len(reminders) == 0 {
		return ""
	}

	// Sort by PostedAt ascending (oldest first)
	sort.Slice(reminders, func(i, j int) bool {
		return reminders[i].PostedAt.Before(reminders[j].PostedAt)
	})

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var b strings.Builder
	b.WriteString("レビューリマインド\n")

	var currentDate string
	for _, r := range reminders {
		dateKey := r.PostedAt.Format("1/2")
		if dateKey != currentDate {
			currentDate = dateKey
			daysElapsed := int(math.Floor(today.Sub(time.Date(r.PostedAt.Year(), r.PostedAt.Month(), r.PostedAt.Day(), 0, 0, 0, 0, r.PostedAt.Location())).Hours() / 24))
			b.WriteString(fmt.Sprintf("\n%s(%d日経過)\n", dateKey, daysElapsed))
		}
		if len(r.Mentions) > 0 {
			b.WriteString(strings.Join(r.Mentions, " ") + "\n")
		}
		b.WriteString(fmt.Sprintf("<%s|%s/%s#%d> - %s / %s\n", r.URL, r.Owner, r.Repo, r.Number, r.Title, r.StatusText))
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
	PostedAt   time.Time
}
