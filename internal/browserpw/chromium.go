// SPDX-License-Identifier: GPL-3.0-or-later

// Il package browserpw recupera le password salvate nei browser basati su
// Chromium (Chrome, Edge) decifrandole dai file di profilo. È pensato per
// funzionare offline, leggendo i file da un disco montato, senza chiamare le
// API di Windows: la catena DPAPI/ABE viene applicata ai dati estratti.
package browserpw

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// dpapiPrefix precede la chiave AES cifrata dentro il file "Local State".
var dpapiPrefix = []byte("DPAPI")

// Prefissi dei valori password nel database dei login di Chromium.
var (
	prefixV10 = []byte("v10")
	prefixV11 = []byte("v11")
)

// ErrLegacyDPAPIValue indica un valore password senza prefisso "v10"/"v11":
// è un blob DPAPI diretto (schema vecchio), da decifrare con la catena DPAPI,
// non con AES-GCM.
var ErrLegacyDPAPIValue = errors.New("valore cifrato in formato DPAPI legacy, non AES-GCM")

// EncryptedKeyFromLocalState estrae dal contenuto del file "Local State" il
// blob che protegge la chiave AES del profilo (campo os_crypt.encrypted_key),
// già decodificato da base64 e privato del prefisso "DPAPI". Il blob restituito
// va poi decifrato con la catena DPAPI (masterkey dell'utente) per ottenere la
// chiave AES a 256 bit usata da DecryptPassword.
func EncryptedKeyFromLocalState(localState []byte) ([]byte, error) {
	var ls struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(localState, &ls); err != nil {
		return nil, fmt.Errorf("lettura di Local State: %w", err)
	}
	if ls.OSCrypt.EncryptedKey == "" {
		return nil, errors.New("Local State senza os_crypt.encrypted_key")
	}
	raw, err := base64.StdEncoding.DecodeString(ls.OSCrypt.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decodifica base64 della chiave: %w", err)
	}
	if !bytes.HasPrefix(raw, dpapiPrefix) {
		return nil, errors.New("chiave di Local State senza prefisso DPAPI")
	}
	return raw[len(dpapiPrefix):], nil
}

// DecryptPassword decifra un valore password_value del database dei login con
// la chiave AES del profilo (32 byte). Gestisce i valori con prefisso
// "v10"/"v11", cioè AES-256-GCM con la struttura:
//
//	3 byte prefisso | 12 byte nonce | ciphertext | 16 byte tag
//
// I valori più vecchi (blob DPAPI diretto, senza prefisso) restituiscono
// ErrLegacyDPAPIValue e vanno decifrati con la catena DPAPI.
func DecryptPassword(aesKey, value []byte) ([]byte, error) {
	if len(value) < 3 || !(bytes.Equal(value[:3], prefixV10) || bytes.Equal(value[:3], prefixV11)) {
		return nil, ErrLegacyDPAPIValue
	}
	const (
		nonceLen = 12
		tagLen   = 16
	)
	if len(value) < 3+nonceLen+tagLen {
		return nil, errors.New("valore cifrato troppo corto")
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("chiave AES non valida: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := value[3 : 3+nonceLen]
	// ciphertext include il tag di autenticazione in coda: gcm.Open lo verifica.
	ciphertext := value[3+nonceLen:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decifratura AES-GCM fallita: %w", err)
	}
	return plain, nil
}
