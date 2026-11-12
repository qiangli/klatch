package databag

import (
	"databag/internal/store"
	"github.com/gorilla/mux"
	"net/http"
	"testing"
	"time"
)

func teamLogin(t *testing.T, user string, role string) *store.Account {
	r, w, _ := NewRequest("POST", "/account/sso", nil)
	signOutpost(r, ssoTestSecret, user, role, ssoTestPrefix, time.Now().Unix())
	SetSSOAccess(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("sso %s: got %d", user, w.Code)
	}
	var identity store.SSOIdentity
	if err := store.DB.Where("email = ?", user).First(&identity).Error; err != nil {
		t.Fatalf("no identity for %s: %v", user, err)
	}
	var account store.Account
	if err := store.DB.Where("id = ?", identity.AccountID).First(&account).Error; err != nil {
		t.Fatalf("no account for %s: %v", user, err)
	}
	return &account
}

func teamMembers(t *testing.T) int64 {
	slotID := getStrConfigValue(CNFSSOTeamChannel, "")
	var slot store.ChannelSlot
	if err := store.DB.Preload("Channel").Where("channel_slot_id = ? AND channel_id > 0", slotID).First(&slot).Error; err != nil || slot.Channel == nil {
		t.Fatalf("team topic %q not found: %v", slotID, err)
	}
	var count int64
	store.DB.Model(&store.Member{}).Where("channel_id = ?", slot.Channel.ID).Count(&count)
	return count
}

func resetRoster(t *testing.T) {
	t.Helper()
	store.DB.Where("1 = 1").Delete(&store.SSOIdentity{})
	for _, key := range []string{CNFSSOTeamOwner, CNFSSOTeamChannel, CNFSSORosterOwner, CNFSSORoster} {
		if err := setStrConfigValue(key, ""); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		store.DB.Model(&store.Account{}).Where("id IN (?)", store.DB.Model(&store.SSOIdentity{}).Select("account_id")).Update("searchable", false)
		for _, key := range []string{CNFSSOTeamOwner, CNFSSOTeamChannel, CNFSSORosterOwner, CNFSSORoster} {
			setStrConfigValue(key, "")
		}
	})
}

func pushRoster(t *testing.T, owner string, emails ...string) ssoRosterResult {
	t.Helper()
	r, w, _ := NewRequest("PUT", "/admin/access?token=pass", nil)
	SetAdminAccess(w, r)
	var token string
	if err := ReadResponse(w, &token); err != nil {
		t.Fatal(err)
	}
	r, w, _ = NewRequest("PUT", "/admin/sso/roster?token="+token, ssoRosterRequest{Owner: owner, Emails: emails})
	// Use the registered route without starting a second notification worker.
	router := mux.NewRouter()
	for _, endpoint := range endpoints {
		router.Methods(endpoint.Method).Path(endpoint.Pattern).HandlerFunc(endpoint.HandlerFunc)
	}
	router.ServeHTTP(w, r)
	var result ssoRosterResult
	if err := ReadResponse(w, &result); err != nil {
		t.Fatalf("roster: %d: %v", w.Code, err)
	}
	return result
}

func TestSSOTeam(t *testing.T) {
	t.Setenv(SSOSecretEnv, ssoTestSecret)
	t.Setenv(SSOTeamTopicEnv, "Crew")
	resetRoster(t)
	b := teamLogin(t, "teamb@example.com", "admin")
	if getStrConfigValue(CNFSSOTeamChannel, "") != "" || getStrConfigValue(CNFSSOTeamOwner, "") != "" {
		t.Fatal("login chose a Team owner before roster")
	}
	got := pushRoster(t, "teamowner@example.com", "teamb@example.com", "teamc@example.com", "TEAMB@example.com")
	if got != (ssoRosterResult{Members: 3, Created: 2}) {
		t.Fatalf("first roster: %+v", got)
	}
	owner := teamLogin(t, "teamowner@example.com", "user")
	c := teamLogin(t, "teamc@example.com", "user")
	if getStrConfigValue(CNFSSOTeamOwner, "") != owner.GUID || getStrConfigValue(CNFSSORosterOwner, "") != "teamowner@example.com" {
		t.Fatal("roster owner not persisted")
	}
	for _, pair := range [][2]*store.Account{{owner, b}, {owner, c}, {b, c}} {
		if !teamConnected(pair[0], pair[1]) || !teamConnected(pair[1], pair[0]) {
			t.Fatal("incomplete mesh")
		}
	}
	channel := getStrConfigValue(CNFSSOTeamChannel, "")
	if n := teamMembers(t); n != 2 {
		t.Fatalf("members: %d", n)
	}
	if got := pushRoster(t, "teamowner@example.com", "teamb@example.com", "teamc@example.com"); got != (ssoRosterResult{Members: 3}) {
		t.Fatalf("repeat: %+v", got)
	}
	if got := pushRoster(t, "teamowner@example.com", "teamc@example.com"); got != (ssoRosterResult{Members: 2, Disabled: 1}) {
		t.Fatalf("remove: %+v", got)
	}
	if n := teamMembers(t); n != 1 {
		t.Fatalf("after removal: %d", n)
	}
	for _, a := range []*store.Account{owner, c} {
		if teamConnected(a, b) || teamConnected(b, a) {
			t.Fatal("removed member still connected")
		}
	}
	if code, _ := ssoLogin(t, "teamb@example.com"); code != http.StatusForbidden {
		t.Fatalf("removed login: %d", code)
	}
	if code, _ := ssoLogin(t, "teamstranger@example.com"); code != http.StatusForbidden {
		t.Fatalf("unlisted login: %d", code)
	}
	if got := pushRoster(t, "teamowner@example.com", "teamc@example.com"); got != (ssoRosterResult{Members: 2}) {
		t.Fatalf("repeat removal: %+v", got)
	}
	if got := pushRoster(t, "teamowner@example.com", "teamb@example.com", "teamc@example.com"); got != (ssoRosterResult{Members: 3, Enabled: 1}) {
		t.Fatalf("restore: %+v", got)
	}
	teamLogin(t, "teamb@example.com", "admin")
	if !teamConnected(b, c) || !teamConnected(c, b) || teamMembers(t) != 2 {
		t.Fatal("restored member missing")
	}
	if getStrConfigValue(CNFSSOTeamChannel, "") != channel {
		t.Fatal("channel replaced")
	}
	var sessions int64
	store.DB.Model(&store.Session{}).Where("app_name = ?", ssoTeamApp).Count(&sessions)
	if sessions != 0 {
		t.Fatalf("helper sessions: %d", sessions)
	}
}
