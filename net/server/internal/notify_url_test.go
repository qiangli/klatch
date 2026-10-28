package databag

import "testing"

func TestContactNotificationURLWithBasePath(t *testing.T) {
	want := "https://sat.example/matrix/h/test/app/klatch/contact/channel/revision?contact=guid.token"
	for _, node := range []string{"sat.example/matrix/h/test/app/klatch", "sat.example/matrix/h/test/app/klatch/"} {
		if got := contactNotificationURL(node, "channel/revision", "guid", "token"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
