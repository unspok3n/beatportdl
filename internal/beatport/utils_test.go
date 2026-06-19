package beatport

import (
	"encoding/json"
	"testing"
)

var benchTemplate = "{number}. {artists} - {name} ({mix_name}) [{catalog_number}] {bpm} {key}"

var benchTemplateValues = map[string]string{
	"number":         "03",
	"artists":        "Deadmau5, Kaskade",
	"name":           "I Remember",
	"mix_name":       "Original Mix",
	"catalog_number": "MAU5001",
	"bpm":            "128",
	"key":            "Bbm",
}

func TestParseTemplate(t *testing.T) {
	got := ParseTemplate("{artists} - {name}", map[string]string{
		"artists": "Deadmau5",
		"name":    "Strobe",
	})
	want := "Deadmau5 - Strobe"
	if got != want {
		t.Fatalf("ParseTemplate() = %q, want %q", got, want)
	}

	// Unknown placeholders must be left intact.
	got = ParseTemplate("{name} {unknown}", map[string]string{"name": "Strobe"})
	want = "Strobe {unknown}"
	if got != want {
		t.Fatalf("ParseTemplate() = %q, want %q", got, want)
	}
}

func TestSanitizeForPath(t *testing.T) {
	got := SanitizeForPath("AC/DC \\ Back\tIn  Black")
	want := "ACDC Back In Black"
	if got != want {
		t.Fatalf("SanitizeForPath() = %q, want %q", got, want)
	}
}

func TestSanitizePath(t *testing.T) {
	got := SanitizePath(`a<b>c:d"e|f?g*h`, "")
	want := "abcdefgh"
	if got != want {
		t.Fatalf("SanitizePath() = %q, want %q", got, want)
	}

	got = SanitizePath("hello world", "_")
	want = "hello_world"
	if got != want {
		t.Fatalf("SanitizePath(whitespace) = %q, want %q", got, want)
	}
}

func TestSanitizedStringUnmarshal(t *testing.T) {
	var s SanitizedString
	if err := json.Unmarshal([]byte(`"  Strobe\n\t (Original  Mix) "`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := "Strobe (Original Mix)"
	if s.String() != want {
		t.Fatalf("SanitizedString = %q, want %q", s.String(), want)
	}
}

func BenchmarkParseTemplate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ParseTemplate(benchTemplate, benchTemplateValues)
	}
}

func BenchmarkSanitizePath(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = SanitizePath(`Deadmau5 - I Remember (Original Mix) <feat. Kaskade>`, "")
	}
}

func BenchmarkSanitizeForPath(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = SanitizeForPath(`Deadmau5 / Kaskade \ I Remember`)
	}
}

func BenchmarkSanitizedStringUnmarshal(b *testing.B) {
	data := []byte(`"  Deadmau5 - I Remember\n\t (Original  Mix) "`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var s SanitizedString
		_ = json.Unmarshal(data, &s)
	}
}
