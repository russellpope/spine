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

func TestParseRequiresSpaceAfterColon(t *testing.T) {
	// "key:value" with no space after the colon is not a pair — parity with
	// adr's original "title: "/"status: " prefix rule. Only bare "key:"
	// (empty value, the one disclosed widening) and "key: value" match.
	kv, has := Parse("---\ntitle:NoSpace\n---\n")
	if !has {
		t.Fatal("want has=true")
	}
	if v, ok := kv["title"]; ok {
		t.Fatalf("title:NoSpace must not parse as a pair; got title=%q", v)
	}

	kv, _ = Parse("---\ntitle: X\n---\n")
	if kv["title"] != "X" {
		t.Fatalf(`want title="X"; kv=%v`, kv)
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

func TestUnquoteScalar(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"quoted title"`, "quoted title"},
		{`"colon: and \"escapes\\\""`, `colon: and "escapes\"`},
		{`""`, ""},
		{`plain title`, `plain title`},
		{`legacy: unquoted title`, `legacy: unquoted title`},
		{`"unbalanced`, `"unbalanced`},
		{`"`, `"`},
		{``, ``},
	}
	for _, c := range cases {
		if got := UnquoteScalar(c.in); got != c.want {
			t.Errorf("UnquoteScalar(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
