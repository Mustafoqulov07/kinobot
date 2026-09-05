package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/url"
	"sort"
	"strings"
)

type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	IsPremium    bool   `json:"is_premium,omitempty"`
}

func ValidateInitData(initData string, botToken string) (*TelegramUser, error) {
	if initData == "" {
		return nil, nil
	}
	if botToken == "" {
		log.Println("VALIDATE FAIL: botToken bo'sh")
		return nil, nil
	}

	values, err := url.ParseQuery(initData)
	if err != nil {
		log.Printf("VALIDATE FAIL: url.ParseQuery xatosi: %v", err)
		return nil, nil
	}

	receivedHash := values.Get("hash")
	if receivedHash == "" {
		log.Println("VALIDATE FAIL: hash maydoni topilmadi")
		return nil, nil
	}

	var pairs []string
	for k, vList := range values {
		if k == "hash" {
			continue
		}
		if len(vList) > 0 {
			pairs = append(pairs, k+"="+vList[0])
		}
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// secret_key = HMAC_SHA256("WebAppData", botToken)
	macSecret := hmac.New(sha256.New, []byte("WebAppData"))
	macSecret.Write([]byte(botToken))
	secretKey := macSecret.Sum(nil)

	// calculated_hash = HMAC_SHA256(secretKey, dataCheckString)
	macData := hmac.New(sha256.New, secretKey)
	macData.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(macData.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(calculatedHash)), []byte(strings.ToLower(receivedHash))) {
		log.Printf("VALIDATE FAIL: hash mos kelmadi. calc=%s recv=%s", calculatedHash, receivedHash)
		return nil, nil
	}

	userRaw := values.Get("user")
	if userRaw == "" {
		log.Println("VALIDATE FAIL: user maydoni topilmadi")
		return nil, nil
	}

	var user TelegramUser
	if err := json.Unmarshal([]byte(userRaw), &user); err != nil {
		log.Printf("VALIDATE FAIL: json unmarshal user: %v", err)
		return nil, nil
	}

	return &user, nil
}
