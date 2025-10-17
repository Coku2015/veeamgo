package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/internal/config"
	"github.com/veeamgo/veeamgo/internal/session"
)

var (
	serverTimeZoneMu   sync.RWMutex
	serverTimeLocation *time.Location
)

func currentServerLocation() *time.Location {
	serverTimeZoneMu.RLock()
	loc := serverTimeLocation
	serverTimeZoneMu.RUnlock()
	if loc != nil {
		return loc
	}
	return time.Local
}

func updateServerTimeLocation(loc *time.Location) {
	serverTimeZoneMu.Lock()
	serverTimeLocation = loc
	serverTimeZoneMu.Unlock()
}

func ensureServerTimeLocation(ctx context.Context, httpClient *client.Client) {
	if httpClient == nil {
		return
	}

	serverTimeZoneMu.RLock()
	loc := serverTimeLocation
	serverTimeZoneMu.RUnlock()
	if loc != nil {
		return
	}

	serverTime, err := httpClient.ServerTime(ctx)
	if err != nil {
		return
	}

	zoneName := firstNonEmpty(serverTime.TimeZone, serverTime.TimeZoneDisplayName, serverTime.TimeZoneID)
	if zoneName == "" {
		zoneName = "VBR"
	}

	_, offset := serverTime.Time.Zone()
	if offset == 0 && serverTime.UtcOffsetMinutes != 0 {
		offset = serverTime.UtcOffsetMinutes * 60
	}
	if offset == 0 {
		if parsed, ok := parseOffsetHint(serverTime.TimeZone); ok {
			offset = parsed
		} else if parsed, ok := parseOffsetHint(serverTime.TimeZoneDisplayName); ok {
			offset = parsed
		} else if parsed, ok := parseOffsetHint(serverTime.TimeZoneID); ok {
			offset = parsed
		}
	}
	if offset == 0 {
		_, inferred := serverTime.Time.Zone()
		offset = inferred
	}

	if offset == 0 {
		updateServerTimeLocation(serverTime.Time.Location())
		return
	}

	updateServerTimeLocation(time.FixedZone(zoneName, offset))
}

func formatTimeForDisplay(value time.Time) string {
	loc := currentServerLocation()
	if loc == nil {
		loc = time.Local
	}
	localTime := value.In(loc)
	_, offset := localTime.Zone()
	return fmt.Sprintf("%s (%s)", localTime.Format("2006-01-02 15:04:05"), formatOffset(offset))
}

func formatOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

var offsetPattern = regexp.MustCompile(`([+-]\d{2}):?(\d{2})`)

func parseOffsetHint(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	matches := offsetPattern.FindStringSubmatch(value)
	if len(matches) != 3 {
		return 0, false
	}
	sign := 1
	if strings.HasPrefix(matches[1], "-") {
		sign = -1
	}
	hours, err := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(matches[1], "+"), "-"))
	if err != nil {
		return 0, false
	}
	minutes, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, false
	}
	return sign * (hours*3600 + minutes*60), true
}

func loadConfigAndProfile() (string, *config.Config, string, *config.Profile, error) {
	cfgPath, err := ensureConfigPath(opts.configPath)
	if err != nil {
		return "", nil, "", nil, err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return "", nil, "", nil, err
	}
	profileName := activeProfile(cfg.DefaultProfile)
	profile := cfg.GetProfile(profileName)
	if profile == nil {
		return "", nil, "", nil, fmt.Errorf("profile %q is not configured; add it to %s or run veeamgo login", profileName, cfgPath)
	}
	return cfgPath, cfg, profileName, profile, nil
}

func loadSession(profileName string) (*session.Session, error) {
	manager, err := session.NewManager()
	if err != nil {
		return nil, err
	}
	sess, err := manager.Fetch(profileName)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func newAPIClient(ctx context.Context) (*client.Client, string, error) {
	_, _, profileName, profile, err := loadConfigAndProfile()
	if err != nil {
		return nil, "", err
	}
	manager, err := session.NewManager()
	if err != nil {
		return nil, "", err
	}

	sess, err := manager.Fetch(profileName)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			sess, err = authenticateWithProfile(ctx, profileName, profile, manager)
			if err != nil {
				return nil, "", err
			}
		} else {
			return nil, "", err
		}
	} else {
		refreshed, err := ensureFreshSession(ctx, profile, sess, manager, profileName)
		if err != nil {
			return nil, "", err
		}
		sess = refreshed
	}

	httpClient, err := client.New(profile, sess)
	if err != nil {
		return nil, "", err
	}
	ensureServerTimeLocation(ctx, httpClient)
	return httpClient, profileName, nil
}

func promptForInput(cmd *cobra.Command, prompt string) (string, error) {
	reader := bufio.NewReader(cmd.InOrStdin())
	fmt.Fprintf(cmd.OutOrStdout(), "%s", prompt)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(value), nil
}

func ensureFreshSession(ctx context.Context, profile *config.Profile, sess *session.Session, manager *session.Manager, profileName string) (*session.Session, error) {
	if sess == nil {
		return nil, fmt.Errorf("session is nil")
	}

	if time.Until(sess.ExpiresAt) > time.Minute {
		return sess, nil
	}

	if sess.RefreshToken != "" {
		refreshed, err := client.Refresh(ctx, profile, sess)
		if err == nil {
			if err := manager.Store(profileName, *refreshed); err != nil {
				return nil, err
			}
			return refreshed, nil
		}
	}

	// Refresh failed or refresh token is missing; fall back to password authentication.
	refreshed, err := authenticateWithProfile(ctx, profileName, profile, manager)
	if err != nil {
		return nil, err
	}
	return refreshed, nil
}

func authenticateWithProfile(ctx context.Context, profileName string, profile *config.Profile, manager *session.Manager) (*session.Session, error) {
	password := resolvePassword(profile)
	if password == "" {
		return nil, fmt.Errorf("no cached session for profile %q and password is not configured; run veeamgo login --save-password or set VEEAMGO_PASSWORD", profileName)
	}

	sess, err := client.Authenticate(ctx, client.Credentials{
		BaseURL:  profile.ServerURL,
		Username: profile.Username,
		Password: password,
		Insecure: profile.Insecure,
	})
	if err != nil {
		return nil, err
	}

	if err := manager.Store(profileName, *sess); err != nil {
		return nil, err
	}

	return sess, nil
}

func resolvePassword(profile *config.Profile) string {
	if profile.Password != "" {
		return profile.Password
	}
	if pwd := os.Getenv("VEEAMGO_PASSWORD"); pwd != "" {
		return pwd
	}
	if pwd := os.Getenv("VEEAMGO_PASS"); pwd != "" {
		return pwd
	}
	return ""
}

func formatYAMLBlock(value any) string {
	if value == nil {
		return ""
	}
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Sprintf("\n  <<unable to render config: %v>>", err)
	}
	block := strings.TrimSpace(string(encoded))
	if block == "" {
		return ""
	}
	return "\n" + indentLines(block, "  ")
}

func indentLines(text, indent string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func formatTimestamp(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return formatTimeForDisplay(*t)
}

func formatTimestampValue(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return formatTimeForDisplay(t)
}

func parseTimeFlag(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("invalid time %q (use RFC3339)", value)
	}
	return &ts, nil
}

func parseJSONMap(value string) (map[string]any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	return payload, nil
}

func parseJSONFile(path string) (map[string]any, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read filter file: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	return payload, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func formatBytes(bytesValue int64) string {
	if bytesValue == 0 {
		return "0 B"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	value := float64(bytesValue)
	idx := 0
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}
	return fmt.Sprintf("%.2f %s", value, units[idx])
}

func requireFlag(flag, hint string) error {
	flag = strings.TrimSpace(flag)
	if flag != "" && !strings.HasPrefix(flag, "--") {
		flag = "--" + flag
	}
	if hint = strings.TrimSpace(hint); hint == "" {
		return fmt.Errorf("flag %s is required", flag)
	}
	return fmt.Errorf("flag %s is required (%s)", flag, hint)
}
