package meta

import "testing"

func TestParse(t *testing.T) {
	content := "---\ntitle: My Title\nstatus: Accepted\nempty:\n---\n\nstatus: Body Decoy\n"
	kv, has := Parse(content)
	if !has {
		t.Fatal("want has=true")
	}
	if kv["title"] != "My Title" || kv["status"] != "Accepted" {
		t.Fatalf("kv=%v", kv)
	}
	if v, ok := kv["empty"]; !ok || v != "" {
		t.Fatalf("empty key: ok=%v v=%q", ok, v)
	}
}

func TestParseFirstOccurrenceWins(t *testing.T) {
	kv, _ := Parse("---\ntitle: First\ntitle: Second\n---\n")
	if kv["title"] != "First" {
		t.Fatalf("kv=%v", kv)
	}
}

func TestParseNoBlock(t *testing.T) {
	if kv, has := Parse("# Just a doc\n"); has || kv != nil {
		t.Fatalf("want nil,false; got %v,%v", kv, has)
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Go with stdlib only": "go-with-stdlib-only",
		"qwen 3.6 (27b)!":     "qwen-3-6-27b",
		"日本語":                 "",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q)=%q want %q", in, got, want)
		}
	}
}
