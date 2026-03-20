package mailer

import (
	"net/smtp"
	"strings"
)

// Mailer sends transactional emails.
type Mailer interface {
	Send(to, subject, body string) error
}

// NoopMailer discards all emails (used when SMTP is not configured).
type NoopMailer struct{}

func (n *NoopMailer) Send(to, subject, body string) error { return nil }

// SMTPMailer sends emails via a plain SMTP relay.
type SMTPMailer struct {
	host string
	port string
	user string
	pass string
	from string
}

func New(host, port, user, pass, from string) Mailer {
	if host == "" {
		return &NoopMailer{}
	}
	return &SMTPMailer{host: host, port: port, user: user, pass: pass, from: from}
}

func (m *SMTPMailer) Send(to, subject, body string) error {
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := m.host + ":" + m.port
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
