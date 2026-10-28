package databag

import (
	"databag/internal/store"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSameNodeChannelNotificationBypassesLimitedPrefixProxy(t *testing.T) {
	var localNotifications atomic.Int32
	var localHeadersOrMethodWrong atomic.Bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" || r.Header.Get("Authorization") != "" {
			localHeadersOrMethodWrong.Store(true)
		}
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/contact/channel/revision":
			if r.Method != http.MethodPut {
				localHeadersOrMethodWrong.Store(true)
			}
			localNotifications.Add(1)
			SetChannelRevision(w, r)
		case "/contact/notification":
			if r.Method != http.MethodPost {
				localHeadersOrMethodWrong.Store(true)
			}
			localNotifications.Add(1)
			SetPushEvent(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	forward := httputil.NewSingleHostReverseProxy(backendURL)
	var frontRequests atomic.Int32
	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if frontRequests.Add(1) > 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/prefix")
		forward.ServeHTTP(w, r)
	}))
	defer front.Close()

	var domain store.Config
	if err := store.DB.Where("config_id = ?", CNFDomain).First(&domain).Error; err != nil {
		t.Fatal(err)
	}
	previousDomain := domain.StrValue
	domain.StrValue = strings.TrimPrefix(front.URL, "http://") + "/prefix"
	if err := store.DB.Save(&domain).Error; err != nil {
		t.Fatal(err)
	}
	defer func() {
		domain.StrValue = previousDomain
		store.DB.Save(&domain)
	}()

	// Two accounts and their connected cards share this one server.
	aGUID, aToken, err := addTestAccount("loopbackA")
	if err != nil {
		t.Fatal(err)
	}
	bGUID, bToken, err := addTestAccount("loopbackB")
	if err != nil {
		t.Fatal(err)
	}
	aCardID, err := addTestCard(aToken, bToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := openTestCard(aToken, aCardID); err != nil {
		t.Fatal(err)
	}
	bCardID, err := addTestCard(bToken, aToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := openTestCard(bToken, bCardID); err != nil {
		t.Fatal(err)
	}
	groupID, err := groupTestCard(aToken, aCardID)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []int{http.StatusOK, http.StatusTooManyRequests} {
		resp, err := http.Get(front.URL + "/prefix/health")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("front returned %d, want %d", resp.StatusCode, want)
		}
	}
	var member store.Account
	if err := store.DB.Where("guid = ?", bGUID).First(&member).Error; err != nil {
		t.Fatal(err)
	}
	var before store.Card
	if err := store.DB.Where("account_id = ?", member.GUID).First(&before).Error; err != nil {
		t.Fatal(err)
	}

	SetNotificationBindAddress(backendURL.Host)
	defer SetNotificationBindAddress("")
	params := &ChannelParams{DataType: "note", Data: "group post", Groups: []string{groupID}, Cards: []string{aCardID}}
	var channel Channel
	if err := APITestMsg(AddChannel, http.MethodPost, "/content/channels", nil, params, APPTokenAgent, aToken, &channel, nil); err != nil {
		t.Fatal(err)
	}
	var author store.Account
	if err := store.DB.Where("guid = ?", aGUID).First(&author).Error; err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var observed int64
	for {
		var card store.Card
		if err := store.DB.Where("account_id = ?", member.GUID).First(&card).Error; err != nil {
			t.Fatal(err)
		}
		if card.NotifiedChannel == author.ChannelRevision && localNotifications.Load() >= 2 {
			observed = card.NotifiedChannel
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("member notified_channel = %d, want %d", card.NotifiedChannel, author.ChannelRevision)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := frontRequests.Load(); got != 2 {
		t.Fatalf("front handled %d requests, want 2 (no notification through front)", got)
	}
	if localHeadersOrMethodWrong.Load() {
		t.Fatal("local notifications used federation headers or the wrong method")
	}
	if observed <= before.NotifiedChannel {
		t.Fatalf("member notified_channel did not increase: %d -> %d", before.NotifiedChannel, observed)
	}
	t.Logf("prefix proxy: 1x200, 1x429; front notification requests: 0; loopback notifications: %d; member notified_channel: %d -> %d", localNotifications.Load(), before.NotifiedChannel, observed)
}
