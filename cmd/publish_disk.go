package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/internal/client"
	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

type publishDiskOptions struct {
	restorePointID string
	diskNames      []string
	allowedIPs     []string
	wait           bool
}

func publishDiskStartCmd() *cobra.Command {
	opts := publishDiskOptions{}

	cmd := &cobra.Command{
		Use:   "publishdisk",
		Short: helptext.PublishDiskShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublishDisk(cmd, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.restorePointID, "restore-point-id", "", "Restore point ID to publish")
	cmd.Flags().StringSliceVar(&opts.diskNames, "disk", nil, "Restore point disk name to publish (repeatable)")
	cmd.Flags().StringSliceVar(&opts.allowedIPs, "allowed-ip", nil, "Allowed IP address for iSCSI target mode (repeatable)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the disk publishing session to finish")

	return cmd
}

func runPublishDisk(cmd *cobra.Command, opts *publishDiskOptions) error {
	restorePointID := strings.TrimSpace(opts.restorePointID)
	if restorePointID == "" {
		return requireFlag("--restore-point-id", "provide the restore point identifier")
	}

	const canonicalMode = "ISCSITarget"

	timeout := 5 * time.Minute
	if opts.wait {
		timeout = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	var restorePoint *client.RestorePoint
	if rp, err := httpClient.RestorePoint(ctx, restorePointID); err == nil {
		restorePoint = rp
	}

	spec := map[string]any{
		"restorePointId": restorePointID,
		"type":           canonicalMode,
	}

	if diskNames := trimSlice(opts.diskNames); len(diskNames) > 0 {
		spec["diskNames"] = diskNames
	}

	allowed := trimSlice(opts.allowedIPs)
	if len(allowed) == 0 {
		return requireFlag("--allowed-ip", "provide at least one allowed IP address for iSCSI target mode")
	}
	spec["allowedIps"] = allowed

	session, err := httpClient.PublishBackupContent(ctx, spec)
	if err != nil {
		return fmt.Errorf("publish disks from restore point %s: %w", restorePointID, err)
	}

	waited := false
	mountID := ""
	var mountDetail *client.InstantRecoveryMountDetail
	var stopSpinner func()

	if session != nil {
		maxWait := time.Duration(0)
		if !opts.wait {
			stop := startSpinner(cmd.ErrOrStderr(), "Publishing disk")
			stopSpinner = func() {
				stop()
				fmt.Fprintln(cmd.ErrOrStderr())
			}
			maxWait = 2 * time.Minute
		}

		finalSession, detail, err := awaitPublishResult(ctx, httpClient, session, restorePointID, maxWait)
		if stopSpinner != nil {
			stopSpinner()
		}

		if err != nil {
			if opts.wait {
				return fmt.Errorf("wait for disk publishing session: %w", err)
			}
			session = finalSession
			if finalSession != nil {
				mountID = strings.TrimSpace(finalSession.ResourceID)
			}
		} else {
			session = finalSession
			if detail != nil {
				mountDetail = detail
				mountID = strings.TrimSpace(detail.Summary.ID)
				waited = true
			} else if finalSession == nil || !isSessionInProgress(finalSession.State) {
				if finalSession != nil {
					mountID = strings.TrimSpace(finalSession.ResourceID)
				}
				waited = true
			}
		}
	}

	if waited && mountDetail == nil && mountID != "" {
		if detail, err := httpClient.DataIntegrationMount(ctx, mountID); err == nil {
			mountDetail = detail
		}
	}

	return printDiskPublishResult(cmd, session, restorePoint, restorePointID, canonicalMode, waited, mountID, mountDetail)
}

func printDiskPublishResult(cmd *cobra.Command, session *client.Session, restorePoint *client.RestorePoint, restorePointID, mode string, waited bool, mountID string, mountDetail *client.InstantRecoveryMountDetail) error {
	format := outputFormat()

	rpName := restorePointID
	rpPlatform := ""
	if restorePoint != nil {
		if trimmed := strings.TrimSpace(restorePoint.Name); trimmed != "" {
			rpName = trimmed
		}
		rpPlatform = restorePoint.PlatformName
	}

	if format == "json" {
		status := "publishing started"
		result := ""
		message := ""
		if session == nil {
			status = "publishing completed"
		} else if waited {
			status = "publishing finished"
			if session.Result != nil {
				result = session.Result.Result
				message = session.Result.Message
			}
		}

		payload := map[string]any{
			"restorePoint": map[string]string{
				"id":       restorePointID,
				"name":     rpName,
				"platform": rpPlatform,
			},
			"mode":    mode,
			"status":  status,
			"waited":  waited,
			"mountId": mountID,
		}
		if session != nil {
			payload["session"] = session
		}
		if result != "" {
			payload["result"] = result
		}
		if strings.TrimSpace(message) != "" {
			payload["details"] = message
		}
		if mountDetail != nil {
			payload["mount"] = mountDetail
		}

		return output.Print(format, payload)
	}

	label := fmt.Sprintf("restore point %q", rpName)
	if session == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Disk publishing for %s completed (mode %s).\n", label, mode)
		if mountID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Mount ID: %s\n", mountID)
		}
		return nil
	}

	if waited {
		result := session.State
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Disk publishing for %s finished with result %s (mode %s, session %s).\n", label, result, mode, session.ID)
		if mountDetail != nil {
			if err := output.Print(format, publishedDiskDetailViewFromDetail(mountDetail)); err != nil {
				return err
			}
			return nil
		}
		if mountID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Mount ID: %s\n", mountID)
		}
		if session.Result != nil && strings.TrimSpace(session.Result.Message) != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", session.Result.Message)
		}
		return nil
	}

	if mountID != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started disk publishing for %s (mode %s, session %s, mount %s).\n", label, mode, session.ID, mountID)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started disk publishing for %s (mode %s, session %s).\n", label, mode, session.ID)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
	return nil
}

func startSpinner(w io.Writer, label string) func() {
	if w == nil {
		return func() {}
	}
	frames := []rune{'|', '/', '-', '\\'}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-stop:
				fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", len(label)+4))
				return
			case <-ticker.C:
				fmt.Fprintf(w, "\r%s %c", label, frames[i%len(frames)])
				i++
			}
		}
	}()
	return func() {
		close(stop)
		wg.Wait()
	}
}

func awaitPublishResult(ctx context.Context, httpClient *client.Client, session *client.Session, restorePointID string, maxWait time.Duration) (*client.Session, *client.InstantRecoveryMountDetail, error) {
	waitCtx := ctx
	var cancel context.CancelFunc
	if maxWait > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, maxWait)
		defer cancel()
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	current := session
	mountID := strings.TrimSpace(session.ResourceID)

	sessionDoneAt := time.Time{}
	for {
		select {
		case <-waitCtx.Done():
			return current, nil, waitCtx.Err()
		case <-ticker.C:
			if s, err := httpClient.Session(waitCtx, session.ID); err == nil && s != nil {
				current = s
				if id := strings.TrimSpace(s.ResourceID); id != "" {
					mountID = id
				}
			}

			detail, id := locateMount(waitCtx, httpClient, mountID, session.ID, restorePointID)
			if detail != nil {
				mountID = id
				state := strings.ToLower(strings.TrimSpace(detail.Summary.State))
				if state == "mounted" || state == "mountfailed" || state == "unmounted" || state == "unmountfailed" {
					return current, detail, nil
				}
				if di := detail.Summary.DataIntegration; di != nil {
					if len(di.ServerIPs) > 0 || di.ServerPort != 0 {
						return current, detail, nil
					}
				}
			}

			if current != nil && !isSessionInProgress(current.State) {
				if detail, id := locateMount(waitCtx, httpClient, mountID, session.ID, restorePointID); detail != nil {
					mountID = id
					return current, detail, nil
				}
				if sessionDoneAt.IsZero() {
					sessionDoneAt = time.Now()
				} else if time.Since(sessionDoneAt) > 30*time.Second {
					return current, nil, nil
				}
			} else {
				sessionDoneAt = time.Time{}
			}
		}
	}
}

func locateMount(ctx context.Context, httpClient *client.Client, mountID, sessionID, restorePointID string) (*client.InstantRecoveryMountDetail, string) {
	if id := strings.TrimSpace(mountID); id != "" {
		if detail, err := httpClient.DataIntegrationMount(ctx, id); err == nil {
			return detail, id
		}
	}

	mounts, err := httpClient.DataIntegrationMounts(ctx, 0)
	if err != nil {
		return nil, mountID
	}
	for _, m := range mounts {
		if strings.EqualFold(m.SessionID, sessionID) || (restorePointID != "" && strings.EqualFold(m.RestorePointID, restorePointID)) {
			if detail, err := httpClient.DataIntegrationMount(ctx, m.ID); err == nil {
				return detail, m.ID
			}
		}
	}
	return nil, mountID
}

func isSessionInProgress(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "starting", "working", "stopping", "pausing", "resuming", "waitingtape", "waitingrepository", "waitingslot", "postprocessing", "running", "pending":
		return true
	default:
		return false
	}
}
