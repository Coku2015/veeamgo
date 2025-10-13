package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/veeamgo/veeamgo/internal/paths"
	"github.com/veeamgo/veeamgo/internal/secrets"
)

// ErrNotFound is returned when a profile session does not exist.
var ErrNotFound = errors.New("session not found")

// Session represents an authenticated session token set.
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Manager persists sessions in an encrypted file under the VeeamGo config directory.
type Manager struct {
	mu   sync.Mutex
	path string
}

// NewManager creates a session manager backed by the ~/.veeamgo directory.
func NewManager() (*Manager, error) {
	base, err := paths.BaseDir()
	if err != nil {
		return nil, err
	}
	return &Manager{path: paths.SessionsPath(base)}, nil
}

// Store saves the session for the profile.
func (m *Manager) Store(profile string, s Session) error {
	if profile == "" {
		return errors.New("profile name is required")
	}

	payload, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	encrypted, err := secrets.EncryptString(m.path, string(payload))
	if err != nil {
		return fmt.Errorf("encrypt session: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := m.load()
	if err != nil {
		return err
	}
	entries[profile] = encrypted
	return m.save(entries)
}

// Fetch retrieves the session for the profile.
func (m *Manager) Fetch(profile string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := m.load()
	if err != nil {
		return nil, err
	}

	ciphertext, ok := entries[profile]
	if !ok {
		return nil, ErrNotFound
	}

	decrypted, err := secrets.DecryptString(m.path, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal([]byte(decrypted), &sess); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return &sess, nil
}

// Delete removes the session for the profile.
func (m *Manager) Delete(profile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := m.load()
	if err != nil {
		return err
	}

	if _, ok := entries[profile]; !ok {
		return ErrNotFound
	}
	delete(entries, profile)
	return m.save(entries)
}

// DeleteAll removes all stored sessions.
func (m *Manager) DeleteAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.Remove(m.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove sessions: %w", err)
	}
	return nil
}

func (m *Manager) load() (map[string]string, error) {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return nil, fmt.Errorf("ensure session dir: %w", err)
	}

	data, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sessions: %w", err)
	}

	entries := make(map[string]string)
	if len(data) == 0 {
		return entries, nil
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode sessions: %w", err)
	}
	return entries, nil
}

func (m *Manager) save(entries map[string]string) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("encode sessions: %w", err)
	}

	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write sessions: %w", err)
	}
	if err := os.Rename(tmp, m.path); err != nil {
		return fmt.Errorf("commit sessions: %w", err)
	}
	return nil
}
