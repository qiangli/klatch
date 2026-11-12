package databag

import (
	"crypto/hmac"
	"crypto/sha256"
	"databag/internal/store"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/theckman/go-securerandom"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Single sign-on through the outpost proxy (elephant shaping, branch
// shape/elephant-sso). With "Trust cloudbox identity" on and an SSO secret
// set, outpost stamps Remote-User and signs it:
//
//	X-Outpost-Identity-Sig = hex HMAC-SHA256(secret,
//	  Remote-User \n Remote-Groups \n X-Forwarded-Prefix \n X-Outpost-Identity-Ts)
//
// A verified email maps to one klatch account, created on first visit, and
// the caller receives a one-time access code that the existing
// PUT /account/access login consumes. Requests that did not come through
// the proxy carry no signature and keep the username/password login.

// SSOSecretEnv names the environment variable holding the outpost SSO secret.
const SSOSecretEnv = "KLATCH_SSO_SECRET"

// SSOClockSkew is the accepted age of the outpost identity timestamp (seconds).
const SSOClockSkew = 60

// SSOAccessExpire is the lifetime of the one-time access code (seconds).
const SSOAccessExpire = 60

var ssoHandleInvalid = regexp.MustCompile(`[^a-z0-9._-]+`)

func ssoSecret() []byte {
	return []byte(strings.TrimSpace(os.Getenv(SSOSecretEnv)))
}

// verifyOutpostIdentity returns the lower-cased Remote-User email when the
// outpost signature is valid and fresh.
func verifyOutpostIdentity(r *http.Request, secret []byte, now int64) (string, bool) {
	prefix := r.Header.Get("X-Forwarded-Prefix")
	user := r.Header.Get("Remote-User")
	role := r.Header.Get("Remote-Groups")
	ts := r.Header.Get("X-Outpost-Identity-Ts")
	sigHex := r.Header.Get("X-Outpost-Identity-Sig")
	if len(secret) == 0 || prefix == "" || user == "" || ts == "" || sigHex == "" {
		return "", false
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || t > now+SSOClockSkew || t < now-SSOClockSkew {
		return "", false
	}
	got, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(user + "\n" + role + "\n" + prefix + "\n" + ts))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return "", false
	}
	email := strings.ToLower(strings.TrimSpace(user))
	if !strings.Contains(email, "@") {
		return "", false
	}
	return email, true
}

// ssoHandle derives a username from the local part of an email.
func ssoHandle(email string) string {
	local := strings.ToLower(email)
	if i := strings.Index(local, "@"); i >= 0 {
		local = local[:i]
	}
	local = strings.Trim(ssoHandleInvalid.ReplaceAllString(local, "-"), "-._")
	if len(local) > 32 {
		local = strings.Trim(local[:32], "-._")
	}
	if local == "" {
		local = "user"
	}
	return local
}

func ssoFreeHandle(tx *gorm.DB, base string) (string, error) {
	for i := 1; i < 1000; i++ {
		name := base
		if i > 1 {
			name = base + strconv.Itoa(i)
		}
		var count int64
		if err := tx.Model(&store.Account{}).Where("username = ? OR handle = ?", name, name).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return name, nil
		}
	}
	return "", errors.New("no free username")
}

// ssoAccount finds the account linked to email, creating it on first visit.
func ssoAccount(email string, name string) (*store.Account, error) {
	var identity store.SSOIdentity
	err := store.DB.Where("email = ?", email).First(&identity).Error
	if err == nil {
		var account store.Account
		err = store.DB.Where("id = ?", identity.AccountID).First(&account).Error
		if err == nil {
			return &account, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		// linked account was deleted by the admin; link a fresh one
		if err := store.DB.Delete(&identity).Error; err != nil {
			return nil, err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	privateKey, publicKey, keyType, err := GenerateRsaKeyPair()
	if err != nil {
		return nil, err
	}
	privatePem := ExportRsaPrivateKeyAsPemStr(privateKey)
	publicPem, err := ExportRsaPublicKeyAsPemStr(publicKey)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(publicPem))
	fingerprint := hex.EncodeToString(hash[:])

	// unusable password: SSO accounts sign in through the proxy only
	random, err := securerandom.Bytes(32)
	if err != nil {
		return nil, err
	}
	password, err := bcrypt.GenerateFromPassword(random, bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	path := getStrConfigValue(CNFAssetPath, APPDefaultPath) + "/" + fingerprint
	if err := os.Mkdir(path, os.ModePerm); err != nil {
		return nil, err
	}

	account := store.Account{
		Searchable: true,
		Password:   password,
		GUID:       fingerprint,
	}
	detail := store.AccountDetail{
		Name:       strings.TrimSpace(name),
		PublicKey:  publicPem,
		PrivateKey: privatePem,
		KeyType:    keyType,
	}
	// Pick the username before the transaction so the transaction only
	// writes: a read-then-write SQLite transaction fails with "database is
	// locked" when another connection writes meanwhile, ignoring the busy
	// timeout. A concurrent first visit taking the same name trips the
	// unique index; retry once with a fresh name.
	for attempt := 0; attempt < 2; attempt++ {
		username, ferr := ssoFreeHandle(store.DB, ssoHandle(email))
		if ferr != nil {
			err = ferr
			break
		}
		account.ID, detail.ID = 0, 0
		account.Username = username
		account.Handle = username
		err = store.DB.Transaction(func(tx *gorm.DB) error {
			if res := tx.Create(&detail).Error; res != nil {
				return res
			}
			account.AccountDetailID = detail.ID
			if res := tx.Create(&account).Error; res != nil {
				return res
			}
			return tx.Create(&store.SSOIdentity{Email: email, AccountID: account.ID}).Error
		})
		if err == nil {
			break
		}
	}
	if err != nil {
		os.Remove(path)
		// a concurrent first visit may have linked the email already
		if store.DB.Where("email = ?", email).First(&identity).Error == nil {
			var linked store.Account
			if store.DB.Where("id = ?", identity.AccountID).First(&linked).Error == nil {
				return &linked, nil
			}
		}
		return nil, err
	}
	return &account, nil
}

// SetSSOAccess exchanges a verified outpost identity for a one-time access code
func SetSSOAccess(w http.ResponseWriter, r *http.Request) {
	secret := ssoSecret()
	if len(secret) == 0 {
		ErrResponse(w, http.StatusNotFound, errors.New("single sign-on not configured"))
		return
	}
	email, ok := verifyOutpostIdentity(r, secret, time.Now().Unix())
	if !ok {
		time.Sleep(APPUsernameWait * time.Millisecond)
		ErrResponse(w, http.StatusUnauthorized, errors.New("no verified outpost identity"))
		return
	}

	// Serialize admission with roster updates, including access-code creation.
	ssoTeamLock.Lock()
	defer ssoTeamLock.Unlock()
	if roster := getStrConfigValue(CNFSSORoster, ""); roster != "" {
		var admitted map[string]bool
		if err := json.Unmarshal([]byte(roster), &admitted); err != nil {
			ErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		if !admitted[email] {
			ErrResponse(w, http.StatusForbidden, errors.New("not on roster"))
			return
		}
	}

	account, err := ssoAccount(email, r.Header.Get("Remote-Name"))
	if err != nil {
		ErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if account.Disabled {
		ErrResponse(w, http.StatusForbidden, errors.New("account disabled"))
		return
	}

	// team mode: a failure here must not block the sign-in itself
	if err := ssoTeamJoin(); err != nil {
		ErrMsg(err)
	}

	data, err := securerandom.Bytes(APPResetSize)
	if err != nil {
		ErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	token := hex.EncodeToString(data)
	accountToken := store.AccountToken{
		AccountID: account.ID,
		TokenType: APPTokenReset,
		Token:     token,
		Expires:   time.Now().Unix() + SSOAccessExpire,
	}
	if err := store.DB.Create(&accountToken).Error; err != nil {
		ErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	WriteResponse(w, token)
}
