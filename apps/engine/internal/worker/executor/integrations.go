package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"time"
)

// DiscordExecutor handles Discord webhook messages.
type DiscordExecutor struct {
	client       *http.Client
	defaultToken string
}

// DiscordConfig represents the configuration for a Discord node.
type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`

	// Message content
	Content   string         `json:"content"`
	Username  string         `json:"username"`
	AvatarURL string         `json:"avatar_url"`
	TTS       bool           `json:"tts"`
	Embeds    []DiscordEmbed `json:"embeds"`
}

// DiscordEmbed represents a Discord embed.
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Author      *DiscordEmbedAuthor `json:"author,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Thumbnail   *DiscordEmbedMedia  `json:"thumbnail,omitempty"`
	Image       *DiscordEmbedMedia  `json:"image,omitempty"`
}

// DiscordEmbedFooter represents a footer in an embed.
type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedAuthor represents an author in an embed.
type DiscordEmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedField represents a field in an embed.
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedMedia represents media in an embed.
type DiscordEmbedMedia struct {
	URL string `json:"url"`
}

// NewDiscordExecutor creates a new Discord executor with connection pooling.
func NewDiscordExecutor() *DiscordExecutor {
	// Configure transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}

	// Get default token from environment
	defaultToken := os.Getenv("DISCORD_BOT_TOKEN")

	return &DiscordExecutor{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		defaultToken: defaultToken,
	}
}

func (e *DiscordExecutor) NodeType() string {
	return "discord"
}

func (e *DiscordExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	start := time.Now()
	logs := make([]LogEntry, 0)

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Starting Discord execution for node %s", req.NodeID),
	})

	var config DiscordConfig
	if err := json.Unmarshal(req.Config, &config); err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("failed to parse Discord config: %v", err),
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	if config.WebhookURL == "" {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: "webhook_url is required",
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	if config.Content == "" && len(config.Embeds) == 0 {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: "content or embeds is required",
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	// Build payload
	payload := map[string]interface{}{
		"content": config.Content,
	}
	if config.Username != "" {
		payload["username"] = config.Username
	}
	if config.AvatarURL != "" {
		payload["avatar_url"] = config.AvatarURL
	}
	if config.TTS {
		payload["tts"] = true
	}
	if len(config.Embeds) > 0 {
		payload["embeds"] = config.Embeds
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("failed to marshal payload: %v", err),
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Sending Discord webhook message",
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("failed to create request: %v", err),
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("request failed: %v", err),
				Type:    ErrorTypeRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logs = append(logs, LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("failed to read response body: %v", err),
		})
	}

	if resp.StatusCode == 429 {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: "rate limited by Discord",
				Type:    ErrorTypeRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	if resp.StatusCode >= 400 {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("Discord error: %s", string(respBody)),
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Discord message sent successfully",
	})

	output, _ := json.Marshal(map[string]interface{}{
		"success":     true,
		"status_code": resp.StatusCode,
	})

	return &ExecuteResponse{
		Output:   output,
		Logs:     logs,
		Duration: time.Since(start),
	}, nil
}

// TwilioExecutor handles Twilio SMS messages.
type TwilioExecutor struct {
	client       *http.Client
	accountSid   string
	authToken    string
	defaultFrom  string
}

// TwilioConfig represents the configuration for a Twilio node.
type TwilioConfig struct {
	AccountSID string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`
	From       string `json:"from"`
	To         string `json:"to"`
	Body       string `json:"body"`
	MediaURL   string `json:"media_url"`
}

// NewTwilioExecutor creates a new Twilio executor with connection pooling.
func NewTwilioExecutor() *TwilioExecutor {
	// Configure transport with connection pooling for Twilio API
	transport := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10, // Most calls to api.twilio.com
		MaxConnsPerHost:     20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}

	// Get credentials from environment
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	defaultFrom := os.Getenv("TWILIO_PHONE_NUMBER")

	return &TwilioExecutor{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		accountSid:  accountSid,
		authToken:   authToken,
		defaultFrom: defaultFrom,
	}
}

// WithCredentials sets default credentials.
func (e *TwilioExecutor) WithCredentials(sid, token string) *TwilioExecutor {
	e.accountSid = sid
	e.authToken = token
	return e
}

func (e *TwilioExecutor) NodeType() string {
	return "twilio"
}

func (e *TwilioExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	start := time.Now()
	logs := make([]LogEntry, 0)

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Starting Twilio execution for node %s", req.NodeID),
	})

	var config TwilioConfig
	if err := json.Unmarshal(req.Config, &config); err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("failed to parse Twilio config: %v", err),
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	// Apply defaults
	if config.AccountSID == "" {
		config.AccountSID = e.accountSid
	}
	if config.AuthToken == "" {
		config.AuthToken = e.authToken
	}

	// Validate
	if config.AccountSID == "" || config.AuthToken == "" {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: "account_sid and auth_token are required",
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	if config.From == "" || config.To == "" || config.Body == "" {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: "from, to, and body are required",
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	// Build form data
	formData := fmt.Sprintf("From=%s&To=%s&Body=%s", config.From, config.To, config.Body)
	if config.MediaURL != "" {
		formData += "&MediaUrl=" + config.MediaURL
	}

	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", config.AccountSID)

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Sending SMS to %s", config.To),
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(formData))
	if err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("failed to create request: %v", err),
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.SetBasicAuth(config.AccountSID, config.AuthToken)

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("request failed: %v", err),
				Type:    ErrorTypeRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logs = append(logs, LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("failed to read response body: %v", err),
		})
	}

	if resp.StatusCode >= 400 {
		errorType := ErrorTypeRetryable
		if resp.StatusCode == 400 || resp.StatusCode == 401 {
			errorType = ErrorTypeNonRetryable
		}
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("Twilio error: %s", string(respBody)),
				Type:    errorType,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "SMS sent successfully",
	})

	return &ExecuteResponse{
		Output:   respBody,
		Logs:     logs,
		Duration: time.Since(start),
	}, nil
}

// StorageExecutor handles cloud storage operations.
type StorageExecutor struct{}

// StorageConfig represents the configuration for a storage node.
type StorageConfig struct {
	Provider   string `json:"provider"`  // s3, gcs, azure
	Operation  string `json:"operation"` // upload, download, delete, list
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	Content    string `json:"content"`
	ContentB64 string `json:"content_base64"`

	// Credentials
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
	Endpoint  string `json:"endpoint"`
}

// NewStorageExecutor creates a new storage executor.
func NewStorageExecutor() *StorageExecutor {
	return &StorageExecutor{}
}

func (e *StorageExecutor) NodeType() string {
	return "storage"
}

func (e *StorageExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	start := time.Now()
	logs := make([]LogEntry, 0)

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Starting storage execution for node %s", req.NodeID),
	})

	// TODO: Implement actual cloud storage operations
	// This would require SDK integration for S3, GCS, Azure Blob

	output, err := json.Marshal(map[string]interface{}{
		"status":  "not_implemented",
		"message": "Storage executor requires cloud SDK integration",
	})
	if err != nil {
		return &ExecuteResponse{
			Error: &ExecutionError{
				Message: fmt.Sprintf("failed to marshal output: %v", err),
				Type:    ErrorTypeNonRetryable,
			},
			Logs:     logs,
			Duration: time.Since(start),
		}, nil
	}

	return &ExecuteResponse{
		Output:   output,
		Logs:     logs,
		Duration: time.Since(start),
	}, nil
}
