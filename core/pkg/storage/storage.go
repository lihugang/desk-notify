package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/logger"
)

type Storage struct {
	appDataDir string
}

func NewStorage(appDataDir string) *Storage {
	return &Storage{
		appDataDir: appDataDir,
	}
}

// SaveFile writes the file to AppData, creates a hard link in Documents, and marks it read-only.
func (s *Storage) SaveFile(notifyID, filename string, r io.Reader, className string) (localPath, docPath string, err error) {
	now := time.Now()
	monthDir := now.Format("2006-01")
	dateDir := now.Format("2006-01-02")

	// 1. AppData Primary Storage
	appDataTargetDir := filepath.Join(s.appDataDir, "data", monthDir, notifyID)
	if err := os.MkdirAll(appDataTargetDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create appdata target dir: %w", err)
	}
	localPath = filepath.Join(appDataTargetDir, filename)

	out, err := os.Create(localPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, r); err != nil {
		return "", "", fmt.Errorf("failed to write local file: %w", err)
	}

	// 2. Documents Mirror Storage via Hard Link
	docBase := config.GetDocumentsDir(className)
	docTargetDir := filepath.Join(docBase, dateDir)
	_ = os.MkdirAll(docTargetDir, 0755)

	docPath = filepath.Join(docTargetDir, fmt.Sprintf("%s_%s", now.Format("150405"), filename))

	// Attempt hard link
	if err := os.Link(localPath, docPath); err != nil {
		// Fallback to copy if hard link fails (e.g., cross-device boundary)
		if cpErr := copyFile(localPath, docPath); cpErr != nil {
			logger.Warnf("[DeskNotify] 备份文件至文档目录失败: %v", cpErr)
			docPath = localPath
		}
	}

	// Mark both as Read-Only
	_ = setReadOnly(localPath)
	_ = setReadOnly(docPath)

	return localPath, docPath, nil
}

func CopyFile(src, dst string) error {
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func setReadOnly(path string) error {
	return os.Chmod(path, 0444)
}
