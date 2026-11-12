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

// Single sign-on teams are managed by the operator-pushed roster.
// SSOTeamTopicEnv names the team topic (default "Team").
const SSOTeamTopicEnv = "KLATCH_SSO_TEAM_TOPIC"

// CNFSSOTeamOwner holds the GUID of the account that owns the team topic.
const CNFSSOTeamOwner = "sso_team_owner"

// CNFSSOTeamChannel holds the owner's channel id of the team topic.
const CNFSSOTeamChannel = "sso_team_channel"

const ssoTeamApp = "klatch-team"

var ssoTeamLock sync.Mutex

func ssoTeamTopic() string {
	if topic := strings.TrimSpace(os.Getenv(SSOTeamTopicEnv)); topic != "" {
		return topic
	}
	return "Team"
}

func setStrConfigValue(configID string, value string) error {
	// Assign by column: a struct would skip the zero value, so "" never cleared.
	var config store.Config
	return store.DB.Where(store.Config{ConfigID: configID}).Assign(map[string]interface{}{"str_value": value}).FirstOrCreate(&config).Error
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

// ssoTeamJoin repairs the roster mesh on sign-in. The caller holds ssoTeamLock.
func ssoTeamJoin() error {
	roster := getStrConfigValue(CNFSSORoster, "")
	if roster == "" {
		return nil
	}
	var admitted map[string]bool
	if err := json.Unmarshal([]byte(roster), &admitted); err != nil {
		return err
	}
	emails := make([]string, 0, len(admitted))
	for email := range admitted {
		emails = append(emails, email)
	}
	var members []store.Account
	if err := store.DB.Where("disabled = ? AND id IN (?)", false,
		store.DB.Model(&store.SSOIdentity{}).Select("account_id").Where("email IN ?", emails)).Order("id").Find(&members).Error; err != nil {
		return err
	}
	agents := &teamAgents{tokens: map[string]string{}}
	defer agents.close()
	for i := range members {
		for j := i + 1; j < len(members); j++ {
			if err := teamConnect(agents, &members[i], &members[j]); err != nil {
				return err
			}
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
