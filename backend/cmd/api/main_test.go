package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer bad.Close()

	if got := healthcheck(ok.URL); got != 0 {
		t.Errorf("healthy server: exit %d, want 0", got)
	}
	if got := healthcheck(bad.URL); got != 1 {
		t.Errorf("503 server: exit %d, want 1", got)
	}
	if got := healthcheck("http://127.0.0.1:1/healthz"); got != 1 {
		t.Errorf("no server: exit %d, want 1", got)
	}
}

func TestSplitList(t *testing.T) {
	got := splitList(" http://a , ,http://b,")
	if len(got) != 2 || got[0] != "http://a" || got[1] != "http://b" {
		t.Errorf("splitList = %q", got)
	}
}
