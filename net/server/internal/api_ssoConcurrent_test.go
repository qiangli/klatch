package databag

import (
	"databag/internal/store"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// Many members opening the shared app at once (Sprint 563 T3): concurrent
// first visits must each get their own account and all join the team. Before
// the busy-timeout and write-only transaction fixes this failed with
// "database is locked".
func TestSSOConcurrentFirstVisits(t *testing.T) {
	t.Setenv(SSOSecretEnv, ssoTestSecret)
	t.Setenv(SSOTeamEnv, "on")
	store.DB.Where("1 = 1").Delete(&store.SSOIdentity{})
	setStrConfigValue(CNFSSOTeamOwner, "")
	setStrConfigValue(CNFSSOTeamChannel, "")
	t.Cleanup(func() {
		store.DB.Model(&store.Account{}).Where("username LIKE ?", "crowd%").Update("searchable", false)
	})

	const members = 8
	codes := make([]int, members)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < members; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, w, _ := NewRequest("POST", "/account/sso", nil)
			signOutpost(r, ssoTestSecret, fmt.Sprintf("crowd%d@example.com", i), "user", ssoTestPrefix, time.Now().Unix())
			<-start
			SetSSOAccess(w, r)
			codes[i] = w.Code
		}(i)
	}
	close(start)
	wg.Wait()

	for i, code := range codes {
		if code != http.StatusOK {
			t.Errorf("member %d: got %d", i, code)
		}
	}
	var accounts []store.Account
	store.DB.Where("id IN (?)", store.DB.Model(&store.SSOIdentity{}).Select("account_id")).Find(&accounts)
	if len(accounts) != members {
		t.Fatalf("%d accounts for %d members", len(accounts), members)
	}
	seen := map[string]bool{}
	for _, a := range accounts {
		if seen[a.Username] {
			t.Fatalf("duplicate username %s", a.Username)
		}
		seen[a.Username] = true
	}
	// every member is connected with every other one
	for i := range accounts {
		for j := range accounts {
			if i != j && !teamConnected(&accounts[i], &accounts[j]) {
				t.Errorf("%s not connected to %s", accounts[i].Username, accounts[j].Username)
			}
		}
	}
}
