package databag

import (
	"databag/internal/store"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// End to end through team mode: members who only ever signed in with single
// sign-on post in the team topic and read each other's messages, using the
// same contact-token path a Klatch client uses.
func TestSSOTeamMessaging(t *testing.T) {
	t.Setenv(SSOSecretEnv, ssoTestSecret)
	resetRoster(t)
	t.Cleanup(func() {
		store.DB.Model(&store.Account{}).Where("username LIKE ?", "msg%").Update("searchable", false)
	})

	owner := teamLogin(t, "msgowner@example.com", "admin")
	b := teamLogin(t, "msgb@example.com", "user")
	c := teamLogin(t, "msgc@example.com", "user")
	pushRoster(t, "msgowner@example.com", "msgb@example.com", "msgc@example.com")
	channelID := getStrConfigValue(CNFSSOTeamChannel, "")
	if channelID == "" {
		t.Fatal("no team topic")
	}

	agents := &teamAgents{tokens: map[string]string{}}
	defer agents.close()
	contactToken := func(member *store.Account) string {
		t.Helper()
		agent, err := agents.agent(member)
		if err != nil {
			t.Fatal(err)
		}
		card, err := teamCard(member, owner)
		if err != nil {
			t.Fatalf("%s has no card for the topic owner: %v", member.Username, err)
		}
		token, err := getCardToken(agent, card.CardSlot.CardSlotID)
		if err != nil {
			t.Fatalf("%s contact token: %v", member.Username, err)
		}
		return token
	}
	vars := map[string]string{"channelID": channelID}

	// member B posts in the team topic
	subject := &Subject{DataType: "superbasictopic", Data: `{"text":"hello team"}`}
	r, w, _ := NewRequest("POST", "/content/channels/{channelID}/topics?contact="+contactToken(b)+"&confirm=true", subject)
	AddChannelTopic(w, mux.SetURLVars(r, vars))
	var posted Topic
	if err := ReadResponse(w, &posted); err != nil {
		t.Fatalf("member post: %v", err)
	}

	// member C reads it
	r, w, _ = NewRequest("GET", "/content/channels/{channelID}/topics?contact="+contactToken(c), nil)
	GetChannelTopics(w, mux.SetURLVars(r, vars))
	var topics []Topic
	if err := ReadResponse(w, &topics); err != nil {
		t.Fatalf("member read: %v", err)
	}
	for _, tp := range topics {
		if tp.ID != posted.ID {
			continue
		}
		if tp.Data == nil || tp.Data.TopicDetail == nil || !strings.Contains(tp.Data.TopicDetail.Data, "hello team") {
			t.Fatalf("topic %s has no message body: %+v", tp.ID, tp.Data)
		}
		if tp.Data.TopicDetail.GUID != b.GUID {
			t.Fatalf("message author = %s, want %s", tp.Data.TopicDetail.GUID, b.GUID)
		}
		return
	}
	t.Fatalf("member C did not see member B's message among %d topics", len(topics))
}
