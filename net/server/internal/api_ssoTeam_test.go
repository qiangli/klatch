package databag

import (
	"databag/internal/store"
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
	if err := store.DB.Preload("Channel").Where("channel_slot_id = ?", slotID).First(&slot).Error; err != nil || slot.Channel == nil {
		t.Fatalf("team topic %q not found: %v", slotID, err)
	}
	var count int64
	store.DB.Model(&store.Member{}).Where("channel_id = ?", slot.Channel.ID).Count(&count)
	return count
}

func TestSSOTeam(t *testing.T) {
	t.Setenv(SSOSecretEnv, ssoTestSecret)
	t.Setenv(SSOTeamEnv, "on")
	t.Setenv(SSOTeamTopicEnv, "Crew")

	// start from an empty team: earlier tests leave SSO accounts behind
	store.DB.Where("1 = 1").Delete(&store.SSOIdentity{})
	setStrConfigValue(CNFSSOTeamOwner, "")
	setStrConfigValue(CNFSSOTeamChannel, "")
	t.Cleanup(func() {
		store.DB.Model(&store.Account{}).Where("username LIKE ?", "team%").Update("searchable", false)
	})

	owner := teamLogin(t, "teamowner@example.com", "admin")
	if getStrConfigValue(CNFSSOTeamOwner, "") != owner.GUID {
		t.Fatalf("host owner does not own the team topic")
	}
	if n := teamMembers(t); n != 0 {
		t.Fatalf("new topic: %d members, want 0", n)
	}

	b := teamLogin(t, "teamb@example.com", "user")
	c := teamLogin(t, "teamc@example.com", "user")

	for _, pair := range [][2]*store.Account{{owner, b}, {owner, c}, {b, c}} {
		if !teamConnected(pair[0], pair[1]) || !teamConnected(pair[1], pair[0]) {
			t.Errorf("%s and %s are not connected", pair[0].Username, pair[1].Username)
		}
	}
	if n := teamMembers(t); n != 2 {
		t.Fatalf("team topic: %d members, want 2", n)
	}

	// a repeat visit changes nothing and leaves no helper sessions behind
	teamLogin(t, "teamb@example.com", "user")
	if n := teamMembers(t); n != 2 {
		t.Fatalf("after repeat visit: %d members, want 2", n)
	}
	var sessions int64
	store.DB.Model(&store.Session{}).Where("app_name = ?", ssoTeamApp).Count(&sessions)
	if sessions != 0 {
		t.Fatalf("%d team helper sessions left", sessions)
	}

	// team mode off: a new account joins nothing
	t.Setenv(SSOTeamEnv, "off")
	d := teamLogin(t, "teamd@example.com", "user")
	if teamConnected(owner, d) || teamConnected(d, owner) {
		t.Fatalf("team mode off still connected a new account")
	}
}
