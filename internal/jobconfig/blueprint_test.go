package jobconfig

import (
	"reflect"
	"testing"
)

func TestParseYAML(t *testing.T) {
	payload := `
name: Example Job
type: VSphereBackup
schedule:
  daily:
    at: "02:00"
`
	blueprint, err := Parse([]byte(payload))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got := findString(blueprint.Spec, "name"); got != "Example Job" {
		t.Fatalf("unexpected name: %q", got)
	}
	if got := findString(blueprint.Spec, "type"); got != "VSphereBackup" {
		t.Fatalf("unexpected type: %q", got)
	}

	schedule, ok := blueprint.Spec["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("schedule not decoded as map: %#v", blueprint.Spec["schedule"])
	}
	if _, ok := schedule["daily"]; !ok {
		t.Fatalf("daily schedule not present: %#v", schedule)
	}
}

func TestApplyOverrides(t *testing.T) {
	bp := &Blueprint{
		Spec: map[string]any{
			"name": "Original",
			"schedule": map[string]any{
				"daily": map[string]any{"at": "02:00"},
			},
		},
	}

	overrides := map[string]string{
		"name":                 "Updated Name",
		"schedule.daily.at":    "03:30",
		"schedule.retryCount":  "5",
		"feature.enabled":      "true",
		"metadata.labels":      `{"team":"prod"}`,
		"retention.restorePts": "[3, 5]",
	}

	if err := bp.ApplyOverrides(overrides); err != nil {
		t.Fatalf("ApplyOverrides returned error: %v", err)
	}

	if got := findString(bp.Spec, "name"); got != "Updated Name" {
		t.Fatalf("name not overridden, got %q", got)
	}

	schedule := bp.Spec["schedule"].(map[string]any)
	daily := schedule["daily"].(map[string]any)
	if got := daily["at"]; got != "03:30" {
		t.Fatalf("schedule.daily.at mismatch: %v", got)
	}
	if got := schedule["retryCount"]; got != int64(5) {
		t.Fatalf("schedule.retryCount mismatch: %v", got)
	}

	if feat := bp.Spec["feature"].(map[string]any)["enabled"]; feat != true {
		t.Fatalf("feature.enabled mismatch: %v", feat)
	}

	labels := bp.Spec["metadata"].(map[string]any)["labels"]
	expected := map[string]any{"team": "prod"}
	if !reflect.DeepEqual(labels, expected) {
		t.Fatalf("metadata.labels mismatch: %#v", labels)
	}

	retention := bp.Spec["retention"].(map[string]any)["restorePts"]
	expectedArray := []any{float64(3), float64(5)}
	if !reflect.DeepEqual(retention, expectedArray) {
		t.Fatalf("retention.restorePts mismatch: %#v", retention)
	}
}

func TestValidate(t *testing.T) {
	bp := &Blueprint{
		Spec: map[string]any{
			"name": "Job",
			"type": "VSphereBackup",
			"id":   "1234",
		},
	}
	if err := bp.Validate(ModeCreate); err != nil {
		t.Fatalf("validate create returned error: %v", err)
	}
	if err := bp.Validate(ModeUpdate); err != nil {
		t.Fatalf("validate update returned error: %v", err)
	}

	delete(bp.Spec, "id")
	if err := bp.Validate(ModeUpdate); err == nil {
		t.Fatalf("expected error when id missing for update")
	}
}

func TestUpdatePayloadMerges(t *testing.T) {
	existing := map[string]any{
		"id":   "abc",
		"name": "Old Name",
		"type": "VSphereBackup",
		"schedule": map[string]any{
			"daily": map[string]any{"at": "02:00"},
		},
		"status": "Stopped",
	}
	spec := map[string]any{
		"name": "New Name",
		"schedule": map[string]any{
			"daily": map[string]any{"at": "03:30"},
		},
	}

	bp := &Blueprint{Spec: spec}
	payload, err := bp.UpdatePayload(existing)
	if err != nil {
		t.Fatalf("UpdatePayload returned error: %v", err)
	}

	if payload["id"] != "abc" {
		t.Fatalf("expected id to be preserved, got %v", payload["id"])
	}
	if payload["name"] != "New Name" {
		t.Fatalf("expected name override, got %v", payload["name"])
	}

	schedule := payload["schedule"].(map[string]any)
	daily := schedule["daily"].(map[string]any)
	if at := daily["at"]; at != "03:30" {
		t.Fatalf("expected schedule update, got %v", at)
	}

	if _, ok := payload["status"]; ok {
		t.Fatalf("read-only field status should be removed")
	}
}

func TestFromExistingPrunesRuntimeFields(t *testing.T) {
	model := map[string]any{
		"id":     "job-1",
		"name":   "Job",
		"type":   "VSphereBackup",
		"status": "Stopped",
		"nested": map[string]any{
			"lastRun": "2024-10-10T10:00:00Z",
			"value":   "kept",
		},
	}
	bp := FromExisting(model)
	if _, ok := bp.Spec["status"]; ok {
		t.Fatalf("status should have been pruned")
	}
	nested := bp.Spec["nested"].(map[string]any)
	if _, ok := nested["lastRun"]; ok {
		t.Fatalf("nested.lastRun should have been pruned")
	}
	if nested["value"] != "kept" {
		t.Fatalf("nested.value should remain, got %v", nested["value"])
	}
}
