package server

import "testing"

func TestHashContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"simple", "hello"},
		{"multiline", "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashContent(tt.input)
			if len(result) != 16 {
				t.Errorf("expected 16 char hash, got %d", len(result))
			}
		})
	}
}

func TestHashContent_Consistency(t *testing.T) {
	content := "test content"
	hash1 := HashContent(content)
	hash2 := HashContent(content)

	if hash1 != hash2 {
		t.Error("same content should produce same hash")
	}
}

func TestHashContent_Uniqueness(t *testing.T) {
	hash1 := HashContent("content1")
	hash2 := HashContent("content2")

	if hash1 == hash2 {
		t.Error("different content should produce different hashes")
	}
}
