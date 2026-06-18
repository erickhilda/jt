package confluence

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(ts *httptest.Server) *Client {
	return &Client{
		baseURL:    ts.URL,
		authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("me@example.com:tok")),
		http:       ts.Client(),
	}
}

func TestGetPage(t *testing.T) {
	var gotAuth, gotAccept, gotPath, gotBodyFormat string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotPath = r.URL.Path
		gotBodyFormat = r.URL.Query().Get("body-format")
		_, _ = w.Write([]byte(`{
			"id":"12345","status":"current","title":"Design Doc","spaceId":"98765",
			"createdAt":"2026-01-02T10:00:00.000Z",
			"version":{"number":7,"createdAt":"2026-06-15T09:00:00.000Z"},
			"body":{"atlas_doc_format":{"representation":"atlas_doc_format","value":"{\"type\":\"doc\"}"}},
			"_links":{"webui":"/spaces/ENG/pages/12345/Design+Doc","base":"https://acme.atlassian.net/wiki"}
		}`))
	}))
	defer ts.Close()

	page, err := testClient(ts).GetPage("12345")
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if page.ID != "12345" || page.Title != "Design Doc" || page.Status != "current" {
		t.Errorf("unexpected page: %+v", page)
	}
	if page.SpaceID != "98765" {
		t.Errorf("spaceId = %q", page.SpaceID)
	}
	if page.Version == nil || page.Version.Number != 7 {
		t.Errorf("version = %+v", page.Version)
	}
	if page.Body.AtlasDocFormat.Value != `{"type":"doc"}` {
		t.Errorf("body value = %q", page.Body.AtlasDocFormat.Value)
	}
	if page.Links.WebUI != "/spaces/ENG/pages/12345/Design+Doc" {
		t.Errorf("webui = %q", page.Links.WebUI)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Errorf("accept = %q", gotAccept)
	}
	if gotPath != "/wiki/api/v2/pages/12345" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBodyFormat != "atlas_doc_format" {
		t.Errorf("body-format = %q", gotBodyFormat)
	}
}

func TestStatusErrors(t *testing.T) {
	cases := []struct {
		code int
		want error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
	}
	for _, tc := range cases {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.code)
		}))
		_, err := testClient(ts).GetPage("1")
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: got %v, want %v", tc.code, err, tc.want)
		}
		ts.Close()
	}
}
