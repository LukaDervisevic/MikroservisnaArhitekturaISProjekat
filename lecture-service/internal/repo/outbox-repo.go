package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
)

type OutboxRepo struct {
	dir string
}

func NewOutboxRepo(dir string) (*OutboxRepo, error) {
	if dir == "" {
		dir = "outbox"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create outbox dir %s: %w", dir, err)
	}
	return &OutboxRepo{dir: dir}, nil
}

func (r *OutboxRepo) WriteEmail(email model.EmailMessage) error {
	if email.ForceFail {
		return fmt.Errorf("forced failure for email %s (demo)", email.IdempotentKey.String())
	}

	filename := fmt.Sprintf("%s_%d.txt", email.IdempotentKey.String(), time.Now().UnixNano())
	path := filepath.Join(r.dir, filename)

	content := fmt.Sprintf("To: %s\nSubject: %s\n\n%s\n", email.To, email.Subject, email.Body)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write outbox file: %w", err)
	}
	return nil
}
