package cmd

import "testing"

func TestResolvePageRef(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"12345", "12345", false},
		{"  12345  ", "12345", false},
		{"https://acme.atlassian.net/wiki/spaces/ENG/pages/12345/Design+Doc", "12345", false},
		{"https://acme.atlassian.net/wiki/spaces/ENG/pages/12345", "12345", false},
		{"https://acme.atlassian.net/wiki/spaces/ENG/pages/edit-v2/12345", "12345", false},
		{"https://acme.atlassian.net/wiki/spaces/ENG/pages/12345/Title?focusedCommentId=9", "12345", false},
		{"", "", true},
		{"not-a-ref", "", true},
		{"https://acme.atlassian.net/wiki/spaces/ENG/overview", "", true},
	}
	for _, tc := range cases {
		got, err := resolvePageRef(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%q: expected error, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSpaceKeyFromWebUI(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/spaces/ENG/pages/12345/Design+Doc", "ENG"},
		{"/spaces/~712020abc/pages/1/Personal", "~712020abc"},
		{"/some/other/path", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := spaceKeyFromWebUI(tc.in); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPageFileKey(t *testing.T) {
	if got := pageFileKey("ENG", "12345", "Design Doc"); got != "ENG__12345__design-doc" {
		t.Errorf("got %q", got)
	}
	if got := pageFileKey("", "9", "Title"); got != "page__9__title" {
		t.Errorf("empty space: got %q", got)
	}
	if got := pageFileKey("ENG", "9", "   "); got != "ENG__9" {
		t.Errorf("empty slug: got %q", got)
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Design Doc", "design-doc"},
		{"  Spaces  &  Symbols!! ", "spaces-symbols"},
		{"Already-good_slug", "already-good-slug"},
		{"...", ""},
		{"CamelCase 123", "camelcase-123"},
	}
	for _, tc := range cases {
		if got := slugify(tc.in); got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
