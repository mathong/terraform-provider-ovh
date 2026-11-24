package helpers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ovh/go-ovh/ovh"
	"github.com/ovh/terraform-provider-ovh/v2/ovh/ovhwrap"
	"go.uber.org/ratelimit"
)

type paginationTestItem struct {
	Name string `json:"name"`
}

// newTestClient returns an ovhwrap client pointed at the given test server URL.
// Dummy credentials are provided so the go-ovh client signs requests (which also
// triggers a call to /auth/time, served by the test server) without reading any
// ambient OVH configuration.
func newTestClient(t *testing.T, serverURL string) *ovhwrap.Client {
	t.Helper()
	client, err := ovh.NewClient(serverURL, "appKey", "appSecret", "consumerKey")
	if err != nil {
		t.Fatalf("failed to create go-ovh client: %s", err)
	}
	return ovhwrap.NewClient(client, ratelimit.NewUnlimited())
}

func TestGetAllPagesV2_MultiplePages(t *testing.T) {
	const regionPath = "/v2/test/region"
	var regionCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/time":
			fmt.Fprint(w, "1700000000")
		case regionPath:
			atomic.AddInt32(&regionCalls, 1)
			switch r.Header.Get("X-Pagination-Cursor") {
			case "":
				// First page: announce a next cursor.
				w.Header().Set("X-Pagination-Cursor-Next", "page2")
				fmt.Fprint(w, `[{"name":"GRA11"},{"name":"SBG5"}]`)
			case "page2":
				// Last page: no next cursor header.
				fmt.Fprint(w, `[{"name":"DE1"}]`)
			default:
				t.Errorf("unexpected cursor: %q", r.Header.Get("X-Pagination-Cursor"))
				w.WriteHeader(http.StatusBadRequest)
			}
		default:
			t.Errorf("unexpected request path: %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := GetAllPagesV2[paginationTestItem](context.Background(), c, regionPath)
	if err != nil {
		t.Fatalf("GetAllPagesV2 returned an error: %s", err)
	}

	want := []string{"GRA11", "SBG5", "DE1"}
	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d: %+v", len(want), len(got), got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("item %d: expected %q, got %q", i, name, got[i].Name)
		}
	}

	if regionCalls != 2 {
		t.Errorf("expected 2 page requests, got %d", regionCalls)
	}
}

func TestGetAllPagesV2_SinglePage(t *testing.T) {
	const regionPath = "/v2/test/region"
	var regionCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/time":
			fmt.Fprint(w, "1700000000")
		case regionPath:
			atomic.AddInt32(&regionCalls, 1)
			if cursor := r.Header.Get("X-Pagination-Cursor"); cursor != "" {
				t.Errorf("unexpected cursor on single page: %q", cursor)
			}
			// No next cursor header => last (and only) page.
			fmt.Fprint(w, `[{"name":"GRA11"}]`)
		default:
			t.Errorf("unexpected request path: %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := GetAllPagesV2[paginationTestItem](context.Background(), c, regionPath)
	if err != nil {
		t.Fatalf("GetAllPagesV2 returned an error: %s", err)
	}

	if len(got) != 1 || got[0].Name != "GRA11" {
		t.Fatalf("expected single item GRA11, got %+v", got)
	}
	if regionCalls != 1 {
		t.Errorf("expected exactly 1 page request, got %d", regionCalls)
	}
}

func newDeleteWaitTestServer(t *testing.T, path string, statuses ...int) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/time":
			fmt.Fprint(w, "1700000000")
		case path:
			i := atomic.AddInt32(&calls, 1) - 1
			status := statuses[min(int(i), len(statuses)-1)]
			w.WriteHeader(status)
			if status == http.StatusOK {
				fmt.Fprint(w, `{"resourceStatus":"DELETING"}`)
			} else {
				fmt.Fprint(w, `{"class":"Client::Error","message":"error"}`)
			}
		default:
			t.Errorf("unexpected request path: %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func TestWaitForAPIv2ResourceDeleted_WaitsFor404(t *testing.T) {
	const path = "/v2/test/resource/1"
	server, calls := newDeleteWaitTestServer(t, path, http.StatusOK, http.StatusNotFound)

	if err := WaitForAPIv2ResourceDeleted(context.Background(), newTestClient(t, server.URL), path); err != nil {
		t.Fatalf("WaitForAPIv2ResourceDeleted returned an error: %s", err)
	}
	if *calls != 2 {
		t.Errorf("expected 2 GET calls, got %d", *calls)
	}
}

func TestWaitForAPIv2ResourceDeleted_StopsOnClientError(t *testing.T) {
	const path = "/v2/test/resource/1"
	server, calls := newDeleteWaitTestServer(t, path, http.StatusForbidden)

	if err := WaitForAPIv2ResourceDeleted(context.Background(), newTestClient(t, server.URL), path); err == nil {
		t.Fatal("expected an error on 403, got nil")
	}
	if *calls != 1 {
		t.Errorf("expected 1 GET call, got %d", *calls)
	}
}

func TestWaitForAPIv2ResourceStatusReady_ErrorIncludesMessage(t *testing.T) {
	const path = "/v2/test/resource/1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/time" {
			fmt.Fprint(w, "1700000000")
			return
		}
		fmt.Fprint(w, `{"resourceStatus":"ERROR","errorMessage":"use force"}`)
	}))
	t.Cleanup(server.Close)

	err := WaitForAPIv2ResourceStatusReady(context.Background(), newTestClient(t, server.URL), path)
	if err == nil || !strings.Contains(err.Error(), "use force") {
		t.Fatalf("expected error containing the API errorMessage, got %v", err)
	}
}
