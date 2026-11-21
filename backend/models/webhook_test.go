package models

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Gpay-Remit/gpay-remit/encryption"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWebhookDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Webhook{}))
	return db
}

func useWebhookTestEncryptor(t *testing.T) {
	t.Helper()
	enc, err := encryption.NewAESGCMEncryptor([]byte("webhook-secret-test-key-1234567890"))
	require.NoError(t, err)
	original := encryption.GetDefaultEncryptor()
	encryption.SetDefaultEncryptor(enc)
	t.Cleanup(func() { encryption.SetDefaultEncryptor(original) })
}

func rawWebhookSecret(t *testing.T, db *gorm.DB, id uint) string {
	t.Helper()
	var rows []struct {
		Secret string
	}
	require.NoError(t, db.Raw("SELECT secret FROM webhooks WHERE id = ?", id).Scan(&rows).Error)
	require.Len(t, rows, 1)
	return rows[0].Secret
}

func TestWebhookSecretEncryptedAtRestAndDecryptedOnRead(t *testing.T) {
	useWebhookTestEncryptor(t)
	db := setupWebhookDB(t)
	plaintext := "whsec_test_secret_value"

	webhook := Webhook{
		UserID: 1,
		URL:    "https://example.com/webhook",
		Secret: plaintext,
		Events: "payment.completed",
	}
	require.NoError(t, db.Create(&webhook).Error)

	stored := rawWebhookSecret(t, db, webhook.ID)
	assert.NotEqual(t, plaintext, stored)
	assert.True(t, strings.HasPrefix(stored, webhookSecretCipherPrefix))

	var loaded Webhook
	require.NoError(t, db.First(&loaded, webhook.ID).Error)
	assert.Equal(t, plaintext, loaded.Secret)

	loaded.Description = "updated"
	require.NoError(t, db.Save(&loaded).Error)
	updatedStored := rawWebhookSecret(t, db, webhook.ID)
	assert.NotEqual(t, stored, updatedStored)
	assert.True(t, strings.HasPrefix(updatedStored, webhookSecretCipherPrefix))

	serialized, err := json.Marshal(loaded)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), plaintext)
	assert.NotContains(t, string(serialized), "secret")
}

func TestEncryptLegacyWebhookSecrets(t *testing.T) {
	useWebhookTestEncryptor(t)
	db := setupWebhookDB(t)
	plaintext := "whsec_legacy_secret_value"

	require.NoError(t, db.Exec(
		"INSERT INTO webhooks (user_id, url, secret, events, is_active) VALUES (?, ?, ?, ?, ?)",
		1, "https://legacy.example.com/hook", plaintext, "payment.completed", true,
	).Error)

	var webhook Webhook
	require.NoError(t, db.Where("secret = ?", plaintext).First(&webhook).Error)
	assert.Equal(t, plaintext, rawWebhookSecret(t, db, webhook.ID))

	require.NoError(t, EncryptLegacyWebhookSecrets(db))
	encrypted := rawWebhookSecret(t, db, webhook.ID)
	assert.NotEqual(t, plaintext, encrypted)
	assert.True(t, strings.HasPrefix(encrypted, webhookSecretCipherPrefix))

	var loaded Webhook
	require.NoError(t, db.First(&loaded, webhook.ID).Error)
	assert.Equal(t, plaintext, loaded.Secret)

	require.NoError(t, EncryptLegacyWebhookSecrets(db))
	assert.Equal(t, encrypted, rawWebhookSecret(t, db, webhook.ID))
}
