package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SMSProvider sends SMS messages through a specific provider.
type SMSProvider interface {
	SendSMS(ctx context.Context, phone, content string) error
}

// smsNotifier adapts an SMSProvider to the Notifier interface.
type smsNotifier struct {
	provider SMSProvider
}

// NewSMS creates a Notifier that sends SMS via the given provider.
// Message.To is used as the phone number, Message.Body as the content.
func NewSMS(provider SMSProvider) Notifier {
	return &smsNotifier{provider: provider}
}

// Send sends an SMS notification.
func (n *smsNotifier) Send(ctx context.Context, msg Message) error {
	return n.provider.SendSMS(ctx, msg.To, msg.Body)
}

// AliyunSMSConfig holds configuration for the Aliyun SMS provider.
type AliyunSMSConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string // SMS 签名
	TemplateCode    string // SMS 模板编号
	Endpoint        string // 默认 dysmsapi.aliyuncs.com
}

// aliyunSMSProvider implements SMSProvider using Aliyun SMS API.
type aliyunSMSProvider struct {
	config AliyunSMSConfig
	client *http.Client
}

// NewAliyunSMS creates an SMSProvider backed by Aliyun SMS.
func NewAliyunSMS(cfg AliyunSMSConfig) SMSProvider {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://dysmsapi.aliyuncs.com"
	}
	return &aliyunSMSProvider{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendSMS sends an SMS via Aliyun. The content is passed as the "code" template parameter.
//
// NOTE: This is a simplified implementation that sends parameters as a plain POST.
// For production use, integrate the official Aliyun SDK
// (github.com/alibabacloud-go/dysmsapi-20170525/v4) which handles HMAC-SHA1 request
// signing automatically. Alternatively, implement the custom SMSProvider interface
// with your preferred SMS provider.
func (p *aliyunSMSProvider) SendSMS(ctx context.Context, phone, content string) error {
	templateParam, _ := json.Marshal(map[string]string{"code": content})

	params := url.Values{
		"PhoneNumbers":  {phone},
		"SignName":      {p.config.SignName},
		"TemplateCode":  {p.config.TemplateCode},
		"TemplateParam": {string(templateParam)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Endpoint,
		bytes.NewReader([]byte(params.Encode())))
	if err != nil {
		return fmt.Errorf("sms request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sms send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("sms: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sms: aliyun returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("sms: unmarshal response: %w", err)
	}
	if result.Code != "OK" {
		return fmt.Errorf("sms: aliyun error %s: %s", result.Code, result.Message)
	}

	return nil
}

// NoopSMSProvider discards all SMS messages. Useful for development/testing.
type NoopSMSProvider struct{}

// SendSMS does nothing and returns nil.
func (n *NoopSMSProvider) SendSMS(_ context.Context, _, _ string) error { return nil }
