package cmd

import "github.com/veeamgo/veeamgo/internal/client"

type sessionSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	Result    string `json:"result"`
	Message   string `json:"message,omitempty"`
	Progress  int    `json:"progressPercent,omitempty"`
	StartedAt string `json:"creationTime,omitempty"`
	EndedAt   string `json:"endTime,omitempty"`
}

func summarizeSession(sess *client.Session) sessionSummary {
	summary := sessionSummary{
		ID:       sess.ID,
		Name:     sess.Name,
		State:    sess.State,
		Progress: sess.ProgressPercent,
	}
	if sess.Result != nil {
		summary.Result = sess.Result.Result
		summary.Message = sess.Result.Message
	}
	if !sess.CreationTime.IsZero() {
		summary.StartedAt = formatTimestampValue(sess.CreationTime)
	}
	if sess.EndTime != nil && !sess.EndTime.IsZero() {
		summary.EndedAt = formatTimestamp(sess.EndTime)
	}
	return summary
}
