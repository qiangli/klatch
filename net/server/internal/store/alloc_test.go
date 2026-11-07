package store

import (
	"path/filepath"
	"testing"
)

func closeStore(t *testing.T) {
	t.Helper()
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBusyTimeout(t *testing.T) {
	SetPath(t.TempDir(), t.TempDir())
	defer closeStore(t)

	var timeout int
	if err := DB.Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil {
		t.Fatal(err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", timeout)
	}
}

func TestReopenStore(t *testing.T) {
	dir := t.TempDir()
	SetPath(dir, filepath.Join(dir, "scripts"))
	if err := DB.Create(&Config{ConfigID: "reopen-test", StrValue: "persisted"}).Error; err != nil {
		t.Fatal(err)
	}
	closeStore(t)

	SetPath(dir, filepath.Join(dir, "scripts"))
	defer closeStore(t)
	var config Config
	if err := DB.Where("config_id = ?", "reopen-test").First(&config).Error; err != nil {
		t.Fatal(err)
	}
	if config.StrValue != "persisted" {
		t.Fatalf("str_value = %q, want persisted", config.StrValue)
	}
}
