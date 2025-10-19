package apiversion

import "testing"

func TestParseAndCompare(t *testing.T) {
	v, err := Parse("1.2-rev1")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if v.Major != 1 || v.Minor != 2 || v.Revision != 1 {
		t.Fatalf("unexpected parse result: %+v", v)
	}

	less, err := Compare("1.1-rev2", "1.2-rev0")
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if less >= 0 {
		t.Fatalf("expected 1.1-rev2 < 1.2-rev0, got %d", less)
	}

	equal, err := Compare("1.2-rev1", "1.2-rev1")
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if equal != 0 {
		t.Fatalf("expected equality, got %d", equal)
	}
}

func TestSupportedAndNormalize(t *testing.T) {
	if !IsSupported("1.3-rev0") {
		t.Fatalf("expected 1.3-rev0 to be supported")
	}
	if IsSupported("2.0-rev0") {
		t.Fatalf("did not expect 2.0-rev0 to be supported")
	}

	supported := Supported()
	if len(supported) == 0 {
		t.Fatalf("expected supported versions to be non-empty")
	}
	supported[0] = "mutated"
	if Highest() != DefaultVersion {
		t.Fatalf("Highest should not be affected by caller mutations")
	}
}

func TestSuggestByBuild(t *testing.T) {
	version, known, newer := SuggestByBuild("12.3.1.1139")
	if version != "1.2-rev1" || !known || newer {
		t.Fatalf("unexpected suggestion: version=%s known=%t newer=%t", version, known, newer)
	}

	version, known, newer = SuggestByBuild("13.2.0.1234")
	if version != "1.3-rev0" || !known || !newer {
		t.Fatalf("expected newer build indication, got version=%s known=%t newer=%t", version, known, newer)
	}

	version, known, newer = SuggestByBuild("10.0.0.1234")
	if version != "1.0-rev1" || known || newer {
		t.Fatalf("expected fallback for old build, got version=%s known=%t newer=%t", version, known, newer)
	}
}

func TestMergeCandidates(t *testing.T) {
	out := MergeCandidates("1.2-rev0", "1.3-rev0", "1.2-rev0", "custom-ver")
	if len(out) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(out))
	}
	if out[0] != "1.3-rev0" || out[1] != "1.2-rev0" || out[2] != "custom-ver" {
		t.Fatalf("unexpected ordering: %v", out)
	}
}
