package databag

import (
	"databag/internal/store"
	"net/http"
	"testing"
)

func TestAddCardWithoutFederation(t *testing.T) {
	domain := getStrConfigValue(CNFDomain, "")
	t.Cleanup(func() { setStrConfigValue(CNFDomain, domain) })
	a, err := ssoAccount("localcarda@example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ssoAccount("localcardb@example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		store.DB.Model(&store.Account{}).Where("id IN ?", []uint{a.ID, b.ID}).Update("searchable", false)
	})
	agents := &teamAgents{tokens: map[string]string{}}
	defer agents.close()
	aToken, err := agents.agent(a)
	if err != nil {
		t.Fatal(err)
	}
	bToken, err := agents.agent(b)
	if err != nil {
		t.Fatal(err)
	}
	message := func(node string) DataMessage {
		t.Helper()
		if err := setStrConfigValue(CNFDomain, node); err != nil {
			t.Fatal(err)
		}
		r, w, _ := NewRequest("GET", "/profile/message?agent="+bToken, nil)
		GetProfileMessage(w, r)
		var msg DataMessage
		if err := ReadResponse(w, &msg); err != nil {
			t.Fatal(err)
		}
		if err := setStrConfigValue(CNFDomain, ""); err != nil {
			t.Fatal(err)
		}
		return msg
	}
	t.Run("external refused", func(t *testing.T) {
		msg := message("external.example.com")
		r, w, _ := NewRequest("POST", "/contact/cards?agent="+aToken, &msg)
		AddCard(w, r)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("external card: %d", w.Code)
		}
		if _, err := teamCard(a, b); err == nil {
			t.Fatal("refused card persisted")
		}
	})
	t.Run("own node accepted", func(t *testing.T) {
		msg := message("")
		r, w, _ := NewRequest("POST", "/contact/cards?agent="+aToken, &msg)
		AddCard(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("own-node card: %d", w.Code)
		}
		card, err := teamCard(a, b)
		if err != nil || card.Node != "" {
			t.Fatalf("own-node card not stored: %v", err)
		}
	})
}
