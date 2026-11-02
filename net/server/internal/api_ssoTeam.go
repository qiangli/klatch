package databag

import (
	"databag/internal/store"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/theckman/go-securerandom"
	"os"
	"strings"
	"sync"
)

// Team mode (elephant shaping, branch shape/elephant-sso). The satellite
// decides who is on the team: the host owner shares klatch with a member's
// email, cloudbox confirms the person is that email, and outpost signs it.
// With KLATCH_SSO_TEAM=on, every account that signs in that way is
// connected with every other single-sign-on account and added to one team
// topic, owned by the host owner (Remote-Groups "admin") when they are the
// first to sign in, otherwise by the earliest team member. It runs on each
// single sign-on, so accounts created before team mode catch up on their
// next visit. Contacts and topics use klatch's own handlers, exactly as a
// client would, so nothing here changes the contact protocol.

// SSOTeamEnv turns team mode on ("on", "true" or "1").
const SSOTeamEnv = "KLATCH_SSO_TEAM"

// SSOTeamTopicEnv names the team topic (default "Team").
const SSOTeamTopicEnv = "KLATCH_SSO_TEAM_TOPIC"

// CNFSSOTeamOwner holds the GUID of the account that owns the team topic.
const CNFSSOTeamOwner = "sso_team_owner"

// CNFSSOTeamChannel holds the owner's channel id of the team topic.
const CNFSSOTeamChannel = "sso_team_channel"

const ssoTeamApp = "klatch-team"

var ssoTeamLock sync.Mutex

func ssoTeamEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(SSOTeamEnv))) {
	case "on", "true", "1":
		return true
	}
	return false
}

func ssoTeamTopic() string {
	if topic := strings.TrimSpace(os.Getenv(SSOTeamTopicEnv)); topic != "" {
		return topic
	}
	return "Team"
}

func setStrConfigValue(configID string, value string) error {
	var config store.Config
	return store.DB.Where(store.Config{ConfigID: configID}).Assign(store.Config{StrValue: value}).FirstOrCreate(&config).Error
}

// teamAgents mints short-lived sessions so team mode can act as each
// account through the regular handlers; close removes them.
type teamAgents struct {
	tokens   map[string]string
	sessions []uint
}

func (t *teamAgents) agent(account *store.Account) (string, error) {
	if token, ok := t.tokens[account.GUID]; ok {
		return token, nil
	}
	data, err := securerandom.Bytes(APPTokenSize)
	if err != nil {
		return "", err
	}
	access := hex.EncodeToString(data)
	session := &store.Session{AccountID: account.GUID, Token: access, AppName: ssoTeamApp}
	if err := store.DB.Create(session).Error; err != nil {
		return "", err
	}
	t.sessions = append(t.sessions, session.ID)
	t.tokens[account.GUID] = account.GUID + "." + access
	return t.tokens[account.GUID], nil
}

func (t *teamAgents) close() {
	if len(t.sessions) > 0 {
		store.DB.Where("id IN ?", t.sessions).Delete(&store.Session{})
	}
}

// teamCard returns account's card for contact, if any.
func teamCard(account *store.Account, contact *store.Account) (*store.Card, error) {
	var card store.Card
	err := store.DB.Preload("CardSlot").Where("account_id = ? AND guid = ?", account.GUID, contact.GUID).First(&card).Error
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func teamConnected(account *store.Account, contact *store.Account) bool {
	card, err := teamCard(account, contact)
	return err == nil && card.Status == APPCardConnected
}

// teamOpen adds contact to account's cards if missing and asks to connect,
// unless already connected.
func teamOpen(agents *teamAgents, account *store.Account, contact *store.Account) error {
	if teamConnected(account, contact) {
		return nil
	}
	accountAgent, err := agents.agent(account)
	if err != nil {
		return err
	}
	contactAgent, err := agents.agent(contact)
	if err != nil {
		return err
	}
	cardID, err := addTestCard(accountAgent, contactAgent)
	if err != nil {
		return err
	}
	return openTestCard(accountAgent, cardID)
}

// teamConnect makes a and b connected contacts of each other.
func teamConnect(agents *teamAgents, a *store.Account, b *store.Account) error {
	if err := teamOpen(agents, a, b); err != nil {
		return err
	}
	if err := teamOpen(agents, b, a); err != nil {
		return err
	}
	if !teamConnected(a, b) || !teamConnected(b, a) {
		return errors.New("contacts did not connect: " + a.Username + ", " + b.Username)
	}
	return nil
}

// teamTopic makes sure the owner's team topic exists and lists every member.
func teamTopic(agents *teamAgents, owner *store.Account, members []store.Account) error {
	ownerAgent, err := agents.agent(owner)
	if err != nil {
		return err
	}

	var slot store.ChannelSlot
	slotID := getStrConfigValue(CNFSSOTeamChannel, "")
	found := slotID != "" && store.DB.Preload("Channel").Where("account_id = ? AND channel_slot_id = ?", owner.ID, slotID).First(&slot).Error == nil && slot.Channel != nil
	if !found {
		subject, _ := json.Marshal(map[string]string{"subject": ssoTeamTopic()})
		params := &ChannelParams{DataType: "superbasic", Data: string(subject), Groups: []string{}, Cards: []string{}}
		r, w, err := NewRequest("POST", "/content/channels?agent="+ownerAgent, params)
		if err != nil {
			return err
		}
		AddChannel(w, r)
		var channel Channel
		if err := ReadResponse(w, &channel); err != nil {
			return err
		}
		if err := setStrConfigValue(CNFSSOTeamChannel, channel.ID); err != nil {
			return err
		}
		if err := store.DB.Preload("Channel").Where("account_id = ? AND channel_slot_id = ?", owner.ID, channel.ID).First(&slot).Error; err != nil {
			return err
		}
	}

	for i := range members {
		member := &members[i]
		if member.GUID == owner.GUID {
			continue
		}
		card, err := teamCard(owner, member)
		if err != nil || card.Status != APPCardConnected {
			continue
		}
		var count int64
		if err := store.DB.Model(&store.Member{}).Where("channel_id = ? AND card_id = ?", slot.Channel.ID, card.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		r, w, err := NewRequest("PUT", "/content/channels/{channelID}/cards/{cardID}?agent="+ownerAgent, nil)
		if err != nil {
			return err
		}
		r = mux.SetURLVars(r, map[string]string{"channelID": slot.ChannelSlotID, "cardID": card.CardSlot.CardSlotID})
		SetChannelCard(w, r)
		if err := ReadResponse(w, &Channel{}); err != nil {
			return err
		}
	}
	return nil
}

// ssoTeamJoin connects account with the rest of the team and keeps the team
// topic current. isOwner marks the host owner, who owns the team topic.
func ssoTeamJoin(account *store.Account, isOwner bool) error {
	if !ssoTeamEnabled() {
		return nil
	}
	ssoTeamLock.Lock()
	defer ssoTeamLock.Unlock()

	var members []store.Account
	if err := store.DB.Where("disabled = ? AND id IN (?)", false, store.DB.Model(&store.SSOIdentity{}).Select("account_id")).Order("id").Find(&members).Error; err != nil {
		return err
	}

	// The topic owner is chosen once and kept, so the topic never moves:
	// the host owner when they are the first to sign in after team mode is
	// on, otherwise the earliest team member.
	if getStrConfigValue(CNFSSOTeamOwner, "") == "" && len(members) > 0 {
		owner := members[0].GUID
		if isOwner {
			owner = account.GUID
		}
		if err := setStrConfigValue(CNFSSOTeamOwner, owner); err != nil {
			return err
		}
	}

	agents := &teamAgents{tokens: map[string]string{}}
	defer agents.close()

	for i := range members {
		if members[i].GUID == account.GUID {
			continue
		}
		if err := teamConnect(agents, account, &members[i]); err != nil {
			return err
		}
	}

	ownerGUID := getStrConfigValue(CNFSSOTeamOwner, "")
	for i := range members {
		if members[i].GUID == ownerGUID {
			return teamTopic(agents, &members[i], members)
		}
	}
	return nil
}
