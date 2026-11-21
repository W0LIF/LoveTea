package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Sender sends transactional emails via SMTP.
type Sender struct {
	host     string
	port     int
	user     string
	pass     string
	from     string
	useTLS   bool
}

// NewSender creates an SMTP mail sender.
func NewSender(host string, port int, user, pass, from string, useTLS bool) *Sender {
	return &Sender{
		host:   host,
		port:   port,
		user:   user,
		pass:   pass,
		from:   from,
		useTLS: useTLS,
	}
}

// Send sends an email to the given recipient.
func (s *Sender) Send(to, subject, body string) error {
	if s.host == "" {
		return fmt.Errorf("mail: SMTP host not configured")
	}

	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	var tlsConfig *tls.Config
	if s.useTLS {
		tlsConfig = &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}
	}

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", s.from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=\"UTF-8\"",
		"",
		body,
	}, "\r\n")

	if s.useTLS {
		return s.sendTLS(addr, tlsConfig, auth, to, msg)
	}
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}

func (s *Sender) sendTLS(addr string, tlsConfig *tls.Config, auth smtp.Auth, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("mail: tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("mail: smtp client: %w", err)
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("mail: auth: %w", err)
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("mail: from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mail: to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("mail: write: %w", err)
	}
	return w.Close()
}
