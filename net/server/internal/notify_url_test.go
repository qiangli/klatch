package databag

import (
	"net/http"
	"testing"
)

func TestContactNotificationURLWithBasePath(t *testing.T) {
	want := "https://sat.example/matrix/h/test/app/klatch/contact/channel/revision?contact=guid.token"
	for _, node := range []string{"sat.example/matrix/h/test/app/klatch", "sat.example/matrix/h/test/app/klatch/"} {
		if got := contactNotificationURL(node, "channel/revision", "guid", "token"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestFederationHeaders(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req, err := http.NewRequest(method, "https://sat.example/matrix/h/test/app/klatch/contact/channel/revision", nil)
		if err != nil {
			t.Fatal(err)
		}
		setFederationHeaders(req)
		if got := req.Header.Get("Origin"); got != "https://sat.example" {
			t.Errorf("%s origin: %q", method, got)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer klatch-federation" {
			t.Errorf("%s authorization: %q", method, got)
		}
	}
}
