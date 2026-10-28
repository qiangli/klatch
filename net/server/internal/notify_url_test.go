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

func TestNotificationURLSelection(t *testing.T) {
	const module = "channel/revision"
	const bind = "127.0.0.1:7100"
	tests := []struct {
		name, node, domain, want string
		local                    bool
	}{
		{"same domain", "sat.example", "sat.example", "http://127.0.0.1:7100/contact/channel/revision?contact=guid.token", true},
		{"same domain with path", "sat.example/matrix/h/test/app/klatch", "sat.example/matrix/h/test/app/klatch", "http://127.0.0.1:7100/contact/channel/revision?contact=guid.token", true},
		{"trailing slash", "sat.example/matrix/h/test/app/klatch/", "sat.example/matrix/h/test/app/klatch", "http://127.0.0.1:7100/contact/channel/revision?contact=guid.token", true},
		{"remote domain", "remote.example/matrix/h/other/app/klatch", "sat.example/matrix/h/test/app/klatch", "https://remote.example/matrix/h/other/app/klatch/contact/channel/revision?contact=guid.token", false},
		{"different path", "sat.example/matrix/h/other/app/klatch", "sat.example/matrix/h/test/app/klatch", "https://sat.example/matrix/h/other/app/klatch/contact/channel/revision?contact=guid.token", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, local := notificationURL(test.node, test.domain, bind, module, "guid", "token")
			if got != test.want || local != test.local {
				t.Errorf("got (%q, %t), want (%q, %t)", got, local, test.want, test.local)
			}
		})
	}
}

func TestNotificationURLUsesBindHostAndPort(t *testing.T) {
	for bind, want := range map[string]string{
		"127.0.0.1:7101": "http://127.0.0.1:7101/contact/notification?contact=guid.token",
		"[::1]:7100":     "http://[::1]:7100/contact/notification?contact=guid.token",
		":7100":          "http://127.0.0.1:7100/contact/notification?contact=guid.token",
	} {
		got, local := notificationURL("sat.example", "sat.example", bind, "notification", "guid", "token")
		if got != want || !local {
			t.Errorf("bind %q: got (%q, %t), want (%q, true)", bind, got, local, want)
		}
	}
}
