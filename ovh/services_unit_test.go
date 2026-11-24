package ovh

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/go-ovh/ovh"
	"github.com/ovh/terraform-provider-ovh/v2/ovh/ovhwrap"
	"go.uber.org/ratelimit"
)

func TestServiceIdFromResourceNameEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/auth/time" {
			_, _ = w.Write([]byte("0"))
			return
		}
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	client, err := ovh.NewClient(server.URL, "key", "secret", "consumer")
	if err != nil {
		t.Fatal(err)
	}
	c := ovhwrap.NewClient(client, ratelimit.NewUnlimited())

	if _, err := serviceIdFromResourceName(c, "gone"); err == nil {
		t.Error("serviceIdFromResourceName: expected an error, got nil")
	}
	if _, err := serviceIdFromRouteAndResourceName(c, "/route", "gone"); err == nil {
		t.Error("serviceIdFromRouteAndResourceName: expected an error, got nil")
	}
}
