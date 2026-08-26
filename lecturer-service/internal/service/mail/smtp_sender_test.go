package mail

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/google/uuid"
)

// fakeRelay is a minimal SMTP server: enough of the dialogue for one delivery,
// so the sender can be tested without a real relay.
type fakeRelay struct {
	listener net.Listener
	done     chan struct{}

	from string
	rcpt string
	data string
}

func startFakeRelay(t *testing.T) *fakeRelay {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	relay := &fakeRelay{listener: listener, done: make(chan struct{})}

	go func() {
		defer close(relay.done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		relay.serve(conn)
	}()

	t.Cleanup(func() { _ = listener.Close() })
	return relay
}

func (r *fakeRelay) serve(conn net.Conn) {
	reader := bufio.NewReader(conn)
	write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

	write("220 fake relay ready")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimRight(line, "\r\n")

		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			write("250 fake relay")
		case strings.HasPrefix(command, "MAIL FROM:"):
			r.from = strings.TrimPrefix(command, "MAIL FROM:")
			write("250 ok")
		case strings.HasPrefix(command, "RCPT TO:"):
			r.rcpt = strings.TrimPrefix(command, "RCPT TO:")
			write("250 ok")
		case command == "DATA":
			write("354 send it")
			var body strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				body.WriteString(dataLine)
			}
			r.data = body.String()
			write("250 queued")
		case command == "QUIT":
			write("221 bye")
			return
		default:
			write("500 unknown command")
		}
	}
}

func (r *fakeRelay) addr(t *testing.T) (host, port string) {
	t.Helper()
	host, port, err := net.SplitHostPort(r.listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split relay address: %v", err)
	}
	return host, port
}

func TestSMTPSenderDeliversMessage(t *testing.T) {
	relay := startFakeRelay(t)
	host, port := relay.addr(t)

	sender := NewSMTPSender(config.SMTPConfig{
		Host:        host,
		Port:        port,
		TLSMode:     config.TLSNone,
		FromAddress: "no-reply@mais.local",
		FromName:    "MAIS Lecturer Service",
		Timeout:     5 * time.Second,
	})

	email := model.EmailMessage{
		IdempotentKey: uuid.New(),
		To:            "predavac@example.com",
		Subject:       "Novo predavanje zakazano: Programiranje ",
		Body:          "Zdravo,\n\nzakazano ti je novo predavanje.\n",
		EnqueuedAt:    time.Now().UTC(),
	}
	if err := sender.Send(context.Background(), email); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	<-relay.done

	if !strings.Contains(relay.from, "no-reply@mais.local") {
		t.Fatalf("unexpected MAIL FROM: %q", relay.from)
	}
	if !strings.Contains(relay.rcpt, email.To) {
		t.Fatalf("unexpected RCPT TO: %q", relay.rcpt)
	}
	for _, want := range []string{
		"From: \"MAIS Lecturer Service\" <no-reply@mais.local>",
		"To: predavac@example.com",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		email.IdempotentKey.String(),
		"zakazano ti je novo predavanje.",
	} {
		if !strings.Contains(relay.data, want) {
			t.Fatalf("delivered message is missing %q:\n%s", want, relay.data)
		}
	}
	// A subject with diacritics has to travel RFC 2047 encoded, never as raw
	// UTF-8 bytes in the header.
	if !strings.Contains(relay.data, "Subject: =?utf-8?q?") || strings.Contains(relay.data, "č/ć") {
		t.Fatalf("subject was not RFC 2047 encoded:\n%s", relay.data)
	}
}

func TestSMTPSenderRejectsInvalidRecipient(t *testing.T) {
	sender := NewSMTPSender(config.SMTPConfig{
		Host:        "127.0.0.1",
		Port:        "1",
		TLSMode:     config.TLSNone,
		FromAddress: "no-reply@mais.local",
		Timeout:     time.Second,
	})

	err := sender.Send(context.Background(), model.EmailMessage{
		IdempotentKey: uuid.New(),
		To:            "not-an-address",
	})
	if err == nil {
		t.Fatal("expected an invalid recipient to be rejected before dialing")
	}
}
