package databag

import (
	"databag/internal/store"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// The persisted roster includes its owner, even when emails omits them.
const CNFSSORoster = "sso_roster"
const CNFSSORosterOwner = "sso_roster_owner"

type ssoRosterRequest struct {
	Owner  string   `json:"owner"`
	Emails []string `json:"emails"`
}

type ssoRosterResult struct {
	Members  int `json:"members"`
	Created  int `json:"created"`
	Enabled  int `json:"enabled"`
	Disabled int `json:"disabled"`
}

func rosterEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, "@") {
		return "", errors.New("invalid roster email")
	}
	return email, nil
}

// SetSSORoster reconciles SSO accounts only. Password accounts have no
// SSOIdentity and are never selected, including username collisions.
func SetSSORoster(w http.ResponseWriter, r *http.Request) {
	if code, err := ParamSessionToken(r); err != nil {
		ErrResponse(w, code, err)
		return
	}
	var request ssoRosterRequest
	if err := ParseRequest(r, w, &request); err != nil {
		ErrResponse(w, http.StatusBadRequest, err)
		return
	}
	owner, err := rosterEmail(request.Owner)
	if err != nil {
		ErrResponse(w, http.StatusBadRequest, err)
		return
	}
	wanted := map[string]bool{owner: true}
	for _, value := range request.Emails {
		email, err := rosterEmail(value)
		if err != nil {
			ErrResponse(w, http.StatusBadRequest, err)
			return
		}
		wanted[email] = true
	}
	ssoTeamLock.Lock()
	defer ssoTeamLock.Unlock()
	result, err := reconcileSSORoster(owner, wanted)
	if err != nil {
		ErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	WriteResponse(w, result)
}

// Caller holds ssoTeamLock, shared with the complete SSO sign-in operation.
func reconcileSSORoster(owner string, wanted map[string]bool) (*ssoRosterResult, error) {
	result := &ssoRosterResult{Members: len(wanted)}
	var identities []store.SSOIdentity
	if err := store.DB.Find(&identities).Error; err != nil {
		return nil, err
	}
	existing := map[string]bool{}
	for _, identity := range identities {
		existing[identity.Email] = true
	}
	var ownerAccount *store.Account
	for email := range wanted {
		account, err := ssoAccount(email, "")
		if err != nil {
			return nil, err
		}
		if !existing[email] {
			result.Created++
		}
		if account.Disabled {
			if err := store.DB.Model(account).Update("disabled", false).Error; err != nil {
				return nil, err
			}
			result.Enabled++
		}
		if email == owner {
			ownerAccount = account
		}
	}
	// Move the existing channel before removing the former owner's contacts.
	if err := teamMoveOwner(ownerAccount); err != nil {
		return nil, err
	}
	if err := store.DB.Find(&identities).Error; err != nil {
		return nil, err
	}
	var removed []store.Account
	var all []store.Account
	for _, identity := range identities {
		var account store.Account
		if err := store.DB.First(&account, identity.AccountID).Error; err != nil {
			return nil, err
		}
		all = append(all, account)
		if !wanted[identity.Email] {
			removed = append(removed, account)
			if !account.Disabled {
				result.Disabled++
			}
		}
	}
	if err := teamDisconnect(removed, all); err != nil {
		return nil, err
	}
	for _, account := range removed {
		if err := store.DB.Model(&store.Account{}).Where("id = ?", account.ID).Update("disabled", true).Error; err != nil {
			return nil, err
		}
		// Revoke existing sessions and outstanding one-time sign-in codes.
		if err := store.DB.Where("account_id = ?", account.GUID).Delete(&store.Session{}).Error; err != nil {
			return nil, err
		}
		if err := store.DB.Where("account_id = ?", account.ID).Delete(&store.AccountToken{}).Error; err != nil {
			return nil, err
		}
	}
	data, err := json.Marshal(wanted)
	if err != nil {
		return nil, err
	}
	if err := setStrConfigValue(CNFSSORosterOwner, owner); err != nil {
		return nil, err
	}
	if err := setStrConfigValue(CNFSSOTeamOwner, ownerAccount.GUID); err != nil {
		return nil, err
	}
	if err := setStrConfigValue(CNFSSORoster, string(data)); err != nil {
		return nil, err
	}
	if err := ssoTeamJoin(); err != nil {
		return nil, err
	}
	return result, nil
}

// Remove both directions of each departed member's SSO contacts, including
// channel memberships. Works for accounts an admin already disabled.
func teamDisconnect(removed, all []store.Account) error {
	gone := map[string]bool{}
	linked := map[string]bool{}
	for _, account := range removed {
		gone[account.GUID] = true
	}
	for _, account := range all {
		linked[account.GUID] = true
	}
	for _, account := range all {
		var cards []store.Card
		if err := store.DB.Preload("CardSlot").Where("account_id = ?", account.GUID).Find(&cards).Error; err != nil {
			return err
		}
		for _, card := range cards {
			if !linked[card.GUID] || (!gone[account.GUID] && !gone[card.GUID]) {
				continue
			}
			err := store.DB.Transaction(func(tx *gorm.DB) error {
				var memberships []store.Member
				if err := tx.Where("card_id = ?", card.ID).Find(&memberships).Error; err != nil {
					return err
				}
				for _, member := range memberships {
					if err := tx.Model(&store.Channel{}).Where("id = ?", member.ChannelID).Update("detail_revision", gorm.Expr("(SELECT channel_revision + 1 FROM accounts WHERE accounts.id = channels.account_id)")).Error; err != nil {
						return err
					}
					if err := tx.Model(&store.ChannelSlot{}).Where("channel_id = ?", member.ChannelID).Update("revision", gorm.Expr("(SELECT channel_revision + 1 FROM accounts WHERE accounts.id = channel_slots.account_id)")).Error; err != nil {
						return err
					}
				}
				if err := tx.Where("card_id = ?", card.ID).Delete(&store.Member{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&card).Association("Groups").Clear(); err != nil {
					return err
				}
				if err := tx.Delete(&card).Error; err != nil {
					return err
				}
				if err := tx.Model(&store.CardSlot{}).Where("id = ?", card.CardSlot.ID).Updates(map[string]interface{}{
					"card_id": 0, "revision": gorm.Expr("(SELECT card_revision + 1 FROM accounts WHERE accounts.id = card_slots.account_id)"),
				}).Error; err != nil {
					return err
				}
				return tx.Model(&store.Account{}).Where("id = ?", account.ID).Updates(map[string]interface{}{
					"card_revision": gorm.Expr("card_revision + 1"), "channel_revision": gorm.Expr("channel_revision + 1"),
				}).Error
			})
			if err != nil {
				return err
			}
		}
		if err := teamNotify(&account); err != nil {
			return err
		}
	}
	return nil
}

// Reload revisions changed by reconciliation before notifying clients.
func teamNotify(account *store.Account) error {
	if err := store.DB.First(account, account.ID).Error; err != nil {
		return err
	}
	SetStatus(account)
	var cards []store.Card
	if err := store.DB.Where("account_id = ? AND status = ?", account.GUID, APPCardConnected).Find(&cards).Error; err != nil {
		return err
	}
	for _, card := range cards {
		SetContactChannelNotification(account, &card)
	}
	return nil
}

// Transfer the channel, its history and assets instead of replacing it with
// an empty topic. Rebuild membership using the new owner's contact cards.
func teamMoveOwner(owner *store.Account) error {
	channelID := getStrConfigValue(CNFSSOTeamChannel, "")
	if channelID == "" {
		return nil
	}
	var slot store.ChannelSlot
	err := store.DB.Preload("Channel").Where("channel_slot_id = ? AND channel_id > 0", channelID).First(&slot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if slot.Channel == nil || slot.AccountID == owner.ID {
		return nil
	}
	var previous store.Account
	if err := store.DB.First(&previous, slot.AccountID).Error; err != nil {
		return err
	}
	var assets []store.Asset
	if err := store.DB.Where("channel_id = ?", slot.ChannelID).Find(&assets).Error; err != nil {
		return err
	}
	base := getStrConfigValue(CNFAssetPath, APPDefaultPath)
	for _, asset := range assets {
		source, err := os.Open(filepath.Join(base, previous.GUID, asset.AssetID))
		if errors.Is(err, os.ErrNotExist) && asset.Status != APPAssetReady {
			continue
		}
		if err != nil {
			return err
		}
		target, err := os.Create(filepath.Join(base, owner.GUID, asset.AssetID))
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(target, source)
		source.Close()
		closeErr := target.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	err = store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", slot.ChannelID).Delete(&store.Member{}).Error; err != nil {
			return err
		}
		if err := tx.Model(slot.Channel).Association("Groups").Clear(); err != nil {
			return err
		}
		for _, model := range []interface{}{&store.Channel{}, &store.Topic{}, &store.TopicSlot{}, &store.Asset{}, &store.Tag{}, &store.TagSlot{}} {
			query := tx.Model(model).Where("channel_id = ?", slot.ChannelID)
			if _, ok := model.(*store.Channel); ok {
				query = tx.Model(model).Where("id = ?", slot.ChannelID)
			}
			if err := query.Update("account_id", owner.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&store.Channel{}).Where("id = ?", slot.ChannelID).Update("detail_revision", owner.ChannelRevision+1).Error; err != nil {
			return err
		}
		// Leave a tombstone for the previous owner's clients.
		if err := tx.Model(&store.ChannelSlot{}).Where("id = ?", slot.ID).Updates(map[string]interface{}{"channel_id": -1, "revision": previous.ChannelRevision + 1}).Error; err != nil {
			return err
		}
		next := store.ChannelSlot{AccountID: owner.ID, ChannelSlotID: channelID, ChannelID: slot.ChannelID, Revision: owner.ChannelRevision + 1}
		if err := tx.Where("account_id = ? AND channel_slot_id = ?", owner.ID, channelID).Assign(map[string]interface{}{"channel_id": slot.ChannelID, "revision": owner.ChannelRevision + 1}).FirstOrCreate(&next).Error; err != nil {
			return err
		}
		return tx.Model(&store.Account{}).Where("id IN ?", []uint{owner.ID, previous.ID}).Update("channel_revision", gorm.Expr("channel_revision + 1")).Error
	})
	if err == nil {
		if err := teamNotify(&previous); err != nil {
			return err
		}
		if err := teamNotify(owner); err != nil {
			return err
		}
	}
	return err
}
