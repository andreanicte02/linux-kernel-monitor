package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreanicte02/linux-kernel-monitor/kmon"
)

func TestSnapshotEndpoint(t *testing.T) {
	h := routes(&kmon.Reader{Dir: "../../kmon/testdata/proc"}, t.TempDir())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/snapshot", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var s kmon.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.Processes) != 5 {
		t.Fatalf("got %d processes", len(s.Processes))
	}
}

func TestSnapshotEndpointWithoutModule(t *testing.T) {
	h := routes(&kmon.Reader{Dir: t.TempDir()}, t.TempDir())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/snapshot", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestNoKillEndpoint(t *testing.T) {
	h := routes(&kmon.Reader{Dir: "../../kmon/testdata/proc"}, t.TempDir())
	for _, method := range []string{"GET", "POST"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/kill/1", nil))
		if rec.Code == http.StatusOK {
			t.Fatalf("%s /kill/1 returned 200; the API must stay read-only", method)
		}
	}
}
