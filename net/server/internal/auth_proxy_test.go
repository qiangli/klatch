package databag

import (
	"net/http/httptest"
	"testing"
)

func TestRequestBasicAuthThroughProxy(t *testing.T) {
	for _, header := range []string{"Authorization", "Credentials"} {
		r := httptest.NewRequest("POST", "/account/apps", nil)
		r.SetBasicAuth("alice", "pass:word")
		if header == "Credentials" {
			r.Header.Set("Credentials", r.Header.Get("Authorization"))
			r.Header.Del("Authorization")
		}
		user, password, ok := requestBasicAuth(r)
		if !ok || user != "alice" || password != "pass:word" {
			t.Errorf("%s: got %q, %q, %v", header, user, password, ok)
		}
	}
}
