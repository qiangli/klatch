package databag

import (
	"crypto/hmac"
	"crypto/sha256"
	"databag/internal/store"
	"encoding/hex"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const ssoTestSecret = "elephant-test-secret"
const ssoTestPrefix = "/matrix/h/elephant/app/klatch"

func signOutpost(r *http.Request, secret, user, role, prefix string, ts int64) {
	stamp := strconv.FormatInt(ts, 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(user + "\n" + role + "\n" + prefix + "\n" + stamp))
	r.Header.Set("Remote-User", user)
	r.Header.Set("Remote-Groups", role)
	r.Header.Set("X-Forwarded-Prefix", prefix)
	r.Header.Set("X-Outpost-Identity-Ts", stamp)
	r.Header.Set("X-Outpost-Identity-Sig", hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyOutpostIdentity(t *testing.T) {
	now := time.Now().Unix()
	cases := []struct {
		name   string
		sign   func(r *http.Request)
		secret string
		want   string
	}{
		{"valid", func(r *http.Request) { signOutpost(r, ssoTestSecret, "Alice@Example.com", "user", ssoTestPrefix, now) }, ssoTestSecret, "alice@example.com"},
		{"wrong secret", func(r *http.Request) { signOutpost(r, "other", "alice@example.com", "user", ssoTestPrefix, now) }, ssoTestSecret, ""},
		{"stale", func(r *http.Request) { signOutpost(r, ssoTestSecret, "alice@example.com", "user", ssoTestPrefix, now-SSOClockSkew-1) }, ssoTestSecret, ""},
		{"future", func(r *http.Request) { signOutpost(r, ssoTestSecret, "alice@example.com", "user", ssoTestPrefix, now+SSOClockSkew+1) }, ssoTestSecret, ""},
		{"no prefix", func(r *http.Request) { signOutpost(r, ssoTestSecret, "alice@example.com", "user", "", now) }, ssoTestSecret, ""},
		{"not an email", func(r *http.Request) { signOutpost(r, ssoTestSecret, "alice", "user", ssoTestPrefix, now) }, ssoTestSecret, ""},
		{"no secret", func(r *http.Request) { signOutpost(r, "", "alice@example.com", "user", ssoTestPrefix, now) }, "", ""},
		{"tampered user", func(r *http.Request) {
			signOutpost(r, ssoTestSecret, "alice@example.com", "user", ssoTestPrefix, now)
			r.Header.Set("Remote-User", "admin@example.com")
		}, ssoTestSecret, ""},
		{"unsigned", func(r *http.Request) { r.Header.Set("Remote-User", "alice@example.com") }, ssoTestSecret, ""},
	}
	for _, c := range cases {
		r, _, _ := NewRequest("POST", "/account/sso", nil)
		c.sign(r)
		got, ok := verifyOutpostIdentity(r, []byte(c.secret), now)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, ok, c.want)
		}
	}
}

func TestSSOHandle(t *testing.T) {
	cases := map[string]string{
		"Alice.Smith@school.org": "alice.smith",
		"bob+chat@x.io":          "bob-chat",
		"..@x.io":                "user",
		"a_very_long_local_part_that_keeps_going_on@x.io": "a_very_long_local_part_that_keep",
	}
	for email, want := range cases {
		if got := ssoHandle(email); got != want {
			t.Errorf("ssoHandle(%q) = %q, want %q", email, got, want)
		}
	}
}

func ssoLogin(t *testing.T, user string) (int, LoginAccess) {
	r, w, _ := NewRequest("POST", "/account/sso", nil)
	signOutpost(r, ssoTestSecret, user, "user", ssoTestPrefix, time.Now().Unix())
	SetSSOAccess(w, r)
	if w.Code != http.StatusOK {
		return w.Code, LoginAccess{}
	}
	var token string
	if err := ReadResponse(w, &token); err != nil {
		t.Fatalf("read sso token: %v", err)
	}
	r, w, _ = NewRequest("PUT", "/account/access?token="+token+"&appName=test", nil)
	SetAccountAccess(w, r)
	var login LoginAccess
	if err := ReadResponse(w, &login); err != nil {
		t.Fatalf("redeem sso token: %v", err)
	}
	// the code is one-time
	r, w, _ = NewRequest("PUT", "/account/access?token="+token+"&appName=test", nil)
	SetAccountAccess(w, r)
	if w.Code == http.StatusOK {
		t.Fatalf("sso access code was accepted twice")
	}
	return http.StatusOK, login
}

func TestSetSSOAccess(t *testing.T) {
	t.Setenv(SSOSecretEnv, "")
	r, w, _ := NewRequest("POST", "/account/sso", nil)
	signOutpost(r, ssoTestSecret, "ssoalice@example.com", "user", ssoTestPrefix, time.Now().Unix())
	SetSSOAccess(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unconfigured: got %d, want 404", w.Code)
	}

	t.Setenv(SSOSecretEnv, ssoTestSecret)

	r, w, _ = NewRequest("POST", "/account/sso", nil)
	r.Header.Set("Remote-User", "ssoalice@example.com")
	SetSSOAccess(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned: got %d, want 401", w.Code)
	}

	// SSO accounts are searchable like self-registered ones; hide them
	// again so later tests see only their own accounts in the listing,
	// as the test helpers do for theirs.
	t.Cleanup(func() {
		store.DB.Model(&store.Account{}).Where("username LIKE ?", "ssoalice%").Update("searchable", false)
	})

	code, first := ssoLogin(t, "SSOAlice@example.com")
	if code != http.StatusOK || first.GUID == "" || first.AppToken == "" {
		t.Fatalf("first visit: code %d, login %+v", code, first)
	}
	_, again := ssoLogin(t, "ssoalice@example.com")
	if again.GUID != first.GUID {
		t.Fatalf("second visit made a new account: %s != %s", again.GUID, first.GUID)
	}

	_, other := ssoLogin(t, "ssoalice@other.example")
	if other.GUID == first.GUID {
		t.Fatalf("different email reused an account")
	}
	var a, b store.Account
	store.DB.Where("guid = ?", first.GUID).First(&a)
	store.DB.Where("guid = ?", other.GUID).First(&b)
	if a.Username != "ssoalice" || b.Username != "ssoalice2" {
		t.Fatalf("usernames: %q, %q", a.Username, b.Username)
	}

	store.DB.Model(&b).Update("disabled", true)
	if code, _ := ssoLogin(t, "ssoalice@other.example"); code != http.StatusForbidden {
		t.Fatalf("disabled account: got %d, want 403", code)
	}
}
