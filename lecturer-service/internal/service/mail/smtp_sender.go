package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
)

type SMTPSender struct {
	cfg config.SMTPConfig
}

func NewSMTPSender(cfg config.SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Describe() string {
	return fmt.Sprintf("smtp %s (tls=%s, from=%s)", s.cfg.Addr(), s.cfg.TLSMode, s.cfg.FromAddress)
}

func (s *SMTPSender) Send(ctx context.Context, email model.EmailMessage) error {
	if email.ForceFail {
		return fmt.Errorf("forced failure for email %s (demo)", email.IdempotentKey)
	}
	if _, err := mail.ParseAddress(email.To); err != nil {
		return fmt.Errorf("invalid recipient %q: %w", email.To, err)
	}

	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	conn, err := s.dial(ctx)
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp handshake with %s: %w", s.cfg.Addr(), err)
	}
	defer client.Close()

	if s.cfg.TLSMode == config.TLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp server %s does not offer STARTTLS", s.cfg.Addr())
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("starttls with %s: %w", s.cfg.Addr(), err)
		}
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth as %s: %w", s.cfg.Username, err)
		}
	}

	if err := client.Mail(s.cfg.FromAddress); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(email.To); err != nil {
		return fmt.Errorf("smtp RCPT TO %s: %w", email.To, err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := writer.Write(s.buildMessage(email)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write message body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close message body: %w", err)
	}

	return client.Quit()
}

func (s *SMTPSender) dial(ctx context.Context) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: s.cfg.Timeout}

	if s.cfg.TLSMode == config.TLSImplicit {
		conn, err := (&tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{ServerName: s.cfg.Host},
		}).DialContext(ctx, "tcp", s.cfg.Addr())
		if err != nil {
			return nil, fmt.Errorf("tls dial %s: %w", s.cfg.Addr(), err)
		}
		return conn, nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", s.cfg.Addr())
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", s.cfg.Addr(), err)
	}
	return conn, nil
}

func (s *SMTPSender) buildMessage(email model.EmailMessage) []byte {
	from := mail.Address{Name: s.cfg.FromName, Address: s.cfg.FromAddress}

	sentAt := time.Now()
	if !email.EnqueuedAt.IsZero() {
		sentAt = email.EnqueuedAt
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "From: %s\r\n", from.String())
	fmt.Fprintf(&sb, "To: %s\r\n", email.To)
	fmt.Fprintf(&sb, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", email.Subject))
	fmt.Fprintf(&sb, "Date: %s\r\n", sentAt.Format(time.RFC1123Z))
	fmt.Fprintf(&sb, "Message-ID: <%s@%s>\r\n", email.IdempotentKey, messageIDDomain(s.cfg.FromAddress))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(strings.ReplaceAll(email.Body, "\r\n", "\n"))

	return []byte(sb.String())
}

func messageIDDomain(fromAddress string) string {
	if _, domain, ok := strings.Cut(fromAddress, "@"); ok && domain != "" {
		return domain
	}
	return "mais.local"
}
