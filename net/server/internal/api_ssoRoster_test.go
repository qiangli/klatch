package databag

import (
	"bytes"
	"databag/internal/store"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
)

func TestSSORosterPasswordAccountsUntouched(t *testing.T) {
	resetRoster(t)
	guid, _, err := addTestAccount("rosterpassword")
	if err != nil {
		t.Fatal(err)
	}
	var before store.Account
	if err := store.DB.Where("guid = ?", guid).First(&before).Error; err != nil {
		t.Fatal(err)
	}
	// A matching local part must create a separate SSO account.
	got := pushRoster(t, "rosterpassword@example.com")
	if got != (ssoRosterResult{Members: 1, Created: 1}) {
		t.Fatalf("create: %+v", got)
	}
	var after store.Account
	store.DB.First(&after, before.ID)
	if after.Disabled != before.Disabled || !bytes.Equal(after.Password, before.Password) || after.Username != before.Username {
		t.Fatal("password account changed")
	}
	store.DB.Model(&before).Update("disabled", true)
	pushRoster(t, "rosterpassword@example.com")
	store.DB.First(&after, before.ID)
	if !after.Disabled {
		t.Fatal("password account re-enabled")
	}
	pushRoster(t, "rosterother@example.com")
	store.DB.First(&after, before.ID)
	if !after.Disabled || !bytes.Equal(after.Password, before.Password) {
		t.Fatal("password account modified on removal")
	}
}

func TestSSORosterAuthorizationAndValidation(t *testing.T) {
	resetRoster(t)
	r, w, _ := NewRequest("PUT", "/admin/sso/roster?token=invalid", ssoRosterRequest{Owner: "owner@example.com"})
	SetSSORoster(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized: %d", w.Code)
	}
	r, w, _ = NewRequest("PUT", "/admin/access?token=pass", nil)
	SetAdminAccess(w, r)
	var token string
	if err := ReadResponse(w, &token); err != nil {
		t.Fatal(err)
	}
	for _, request := range []ssoRosterRequest{{}, {Owner: "Name <a@x>"}, {Owner: "a@x", Emails: []string{"bad"}}} {
		r, w, _ = NewRequest("PUT", "/admin/sso/roster?token="+token, request)
		SetSSORoster(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid roster: %d", w.Code)
		}
	}
	var count int64
	store.DB.Model(&store.SSOIdentity{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid request created accounts")
	}
}

func TestSSORosterMovesExistingTeam(t *testing.T) {
	t.Setenv(SSOSecretEnv, ssoTestSecret)
	resetRoster(t)
	old := teamLogin(t, "movefirst@example.com", "admin")
	owner := teamLogin(t, "moveowner@example.com", "user")
	// Simulate the old first-login-owned Team with existing message history.
	agents := &teamAgents{tokens: map[string]string{}}
	defer agents.close()
	if err := teamConnect(agents, old, owner); err != nil {
		t.Fatal(err)
	}
	if err := teamTopic(agents, old, []store.Account{*old, *owner}); err != nil {
		t.Fatal(err)
	}
	setStrConfigValue(CNFSSOTeamOwner, old.GUID)
	channelID := getStrConfigValue(CNFSSOTeamChannel, "")
	token, err := agents.agent(old)
	if err != nil {
		t.Fatal(err)
	}
	r, w, _ := NewRequest("POST", "/content/channels/{channelID}/topics?agent="+token,
		&Subject{DataType: "superbasictopic", Data: `{"text":"keep this history"}`})
	AddChannelTopic(w, mux.SetURLVars(r, map[string]string{"channelID": channelID}))
	var posted Topic
	if err := ReadResponse(w, &posted); err != nil {
		t.Fatal(err)
	}
	pushRoster(t, "moveowner@example.com", "movefirst@example.com")
	if getStrConfigValue(CNFSSOTeamOwner, "") != owner.GUID || getStrConfigValue(CNFSSOTeamChannel, "") != channelID {
		t.Fatal("wrong transferred channel")
	}
	var slot store.ChannelSlot
	if err := store.DB.Preload("Channel").Where("account_id = ? AND channel_slot_id = ?", owner.ID, channelID).First(&slot).Error; err != nil {
		t.Fatal(err)
	}
	if slot.Channel == nil || slot.Channel.AccountID != owner.ID || teamMembers(t) != 1 {
		t.Fatal("ownership or membership not transferred")
	}
	var topics []store.Topic
	store.DB.Where("channel_id = ?", slot.Channel.ID).Find(&topics)
	if len(topics) != 1 || topics[0].AccountID != owner.ID || topics[0].GUID != old.GUID {
		t.Fatalf("history lost: %+v", topics)
	}
	// Moving back exercises an existing tombstone rather than a fresh slot.
	pushRoster(t, "movefirst@example.com", "moveowner@example.com")
	if getStrConfigValue(CNFSSOTeamOwner, "") != old.GUID || teamMembers(t) != 1 {
		t.Fatal("return transfer failed")
	}
}
