package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Notification struct {
	ID          string        `json:"id"`
	Content     string        `json:"content"`
	Pinned      bool          `json:"pinned"`
	Duration    int           `json:"duration"`
	Sender      string        `json:"sender"`
	CreatedAt   time.Time     `json:"createdAt"`
	Images      []string      `json:"images"`
	Attachments []*Attachment `json:"attachments"`
}

type Attachment struct {
	ID        string `json:"id"`
	NotifyID  string `json:"notifyId"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `json:"type"`
	LocalPath string `json:"localPath"`
	DocPath   string `json:"docPath"`
	IsImage   bool   `json:"isImage"`
	URL       string `json:"url"`
}

type Store struct {
	db *sql.DB
}

func NewStore(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "desk_notify.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate database schema: %w", err)
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		pinned BOOLEAN NOT NULL DEFAULT 1,
		duration INTEGER NOT NULL,
		sender TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS attachments (
		id TEXT PRIMARY KEY,
		notify_id TEXT NOT NULL,
		name TEXT NOT NULL,
		size INTEGER NOT NULL,
		mime_type TEXT,
		local_path TEXT NOT NULL,
		doc_path TEXT NOT NULL,
		is_image BOOLEAN NOT NULL DEFAULT 0,
		FOREIGN KEY (notify_id) REFERENCES notifications(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_notify_created ON notifications(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_att_notify ON attachments(notify_id);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	_, _ = s.db.Exec(`ALTER TABLE notifications ADD COLUMN pinned BOOLEAN NOT NULL DEFAULT 1`)
	return nil
}

func (s *Store) SaveNotification(n *Notification, atts []*Attachment) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtN, err := tx.Prepare(`
		INSERT INTO notifications (id, content, pinned, duration, sender, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtN.Close()

	if _, err := stmtN.Exec(n.ID, n.Content, n.Pinned, n.Duration, n.Sender, n.CreatedAt); err != nil {
		return err
	}

	if len(atts) > 0 {
		stmtA, err := tx.Prepare(`
			INSERT INTO attachments (id, notify_id, name, size, mime_type, local_path, doc_path, is_image)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmtA.Close()

		for _, a := range atts {
			if _, err := stmtA.Exec(a.ID, n.ID, a.Name, a.Size, a.MimeType, a.LocalPath, a.DocPath, a.IsImage); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *Store) GetHistory(limit, offset int) ([]*Notification, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	rows, err := s.db.Query(`
		SELECT id, content, pinned, duration, sender, created_at
		FROM notifications
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*Notification, 0)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Content, &n.Pinned, &n.Duration, &n.Sender, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		n.Images = make([]string, 0)
		n.Attachments = make([]*Attachment, 0)
		list = append(list, &n)
	}

	s.populateAttachments(list)
	return list, total, nil
}

func (s *Store) populateAttachments(list []*Notification) {
	for _, n := range list {
		attRows, err := s.db.Query(`
			SELECT id, notify_id, name, size, mime_type, local_path, doc_path, is_image
			FROM attachments
			WHERE notify_id = ?
		`, n.ID)
		if err != nil {
			continue
		}
		for attRows.Next() {
			var a Attachment
			if err := attRows.Scan(&a.ID, &a.NotifyID, &a.Name, &a.Size, &a.MimeType, &a.LocalPath, &a.DocPath, &a.IsImage); err == nil {
				a.URL = fmt.Sprintf("/api/files/%s", a.ID)
				if a.IsImage {
					n.Images = append(n.Images, a.URL)
				} else {
					n.Attachments = append(n.Attachments, &a)
				}
			}
		}
		attRows.Close()
	}
}

func (s *Store) GetRecentWithinMinutes(minutes int) ([]*Notification, error) {
	if minutes <= 0 {
		minutes = 90
	}
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)

	rows, err := s.db.Query(`
		SELECT id, content, pinned, duration, sender, created_at
		FROM notifications
		WHERE created_at >= ?
		ORDER BY created_at DESC
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Notification, 0)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Content, &n.Pinned, &n.Duration, &n.Sender, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.Images = make([]string, 0)
		n.Attachments = make([]*Attachment, 0)
		list = append(list, &n)
	}

	s.populateAttachments(list)
	return list, nil
}

func (s *Store) GetRecent(limit int) ([]*Notification, error) {
	list, _, err := s.GetHistory(limit, 0)
	return list, err
}

func (s *Store) GetAttachment(id string) (*Attachment, error) {
	row := s.db.QueryRow(`
		SELECT id, notify_id, name, size, mime_type, local_path, doc_path, is_image
		FROM attachments
		WHERE id = ?
	`, id)

	var a Attachment
	if err := row.Scan(&a.ID, &a.NotifyID, &a.Name, &a.Size, &a.MimeType, &a.LocalPath, &a.DocPath, &a.IsImage); err != nil {
		return nil, err
	}
	a.URL = fmt.Sprintf("/api/files/%s", a.ID)
	return &a, nil
}
