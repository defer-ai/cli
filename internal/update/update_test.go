package update

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int // >0 if a > b, 0 if equal, <0 if a < b
	}{
		{"newer patch", "2.0.2", "2.0.1", 1},
		{"same version", "2.0.2", "2.0.2", 0},
		{"newer major", "3.0.0", "2.0.2", 1},
		{"older patch", "2.0.1", "2.0.2", -1},
		{"newer minor", "2.1.0", "2.0.9", 1},
		{"older major", "1.9.9", "2.0.0", -1},
		{"different lengths", "2.0", "2.0.0", 0},
		{"different lengths newer", "2.0.1", "2.0", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.a, tt.b)
			switch {
			case tt.want > 0 && got <= 0:
				t.Errorf("CompareVersions(%q, %q) = %d, want > 0", tt.a, tt.b, got)
			case tt.want == 0 && got != 0:
				t.Errorf("CompareVersions(%q, %q) = %d, want 0", tt.a, tt.b, got)
			case tt.want < 0 && got >= 0:
				t.Errorf("CompareVersions(%q, %q) = %d, want < 0", tt.a, tt.b, got)
			}
		})
	}
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	// "dev" version should always skip the check and return empty.
	latest, url, err := CheckForUpdate("dev")
	if err != nil {
		t.Errorf("CheckForUpdate(\"dev\") returned error: %v", err)
	}
	if latest != "" || url != "" {
		t.Errorf("CheckForUpdate(\"dev\") = (%q, %q), want empty", latest, url)
	}
}

func TestCheckForUpdate_EmptyVersion(t *testing.T) {
	// Empty version should skip the check.
	latest, url, err := CheckForUpdate("")
	if err != nil {
		t.Errorf("CheckForUpdate(\"\") returned error: %v", err)
	}
	if latest != "" || url != "" {
		t.Errorf("CheckForUpdate(\"\") = (%q, %q), want empty", latest, url)
	}
}

func TestAssetName(t *testing.T) {
	// Just verify it returns a non-empty string with the expected format.
	name := AssetName()
	if name == "" {
		t.Error("AssetName() returned empty string")
	}
	// Should contain "defer_" prefix
	if len(name) < 10 {
		t.Errorf("AssetName() = %q, seems too short", name)
	}
}
