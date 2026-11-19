// SPDX-License-Identifier: GPL-3.0-or-later

package browserpw

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
)

func TestEncryptedKeyFromLocalState(t *testing.T) {
	// Costruisce un Local State con una chiave "DPAPI" + payload noto.
	payload := []byte{0x01, 0x02, 0x03, 0xAA, 0xBB}
	raw := append([]byte("DPAPI"), payload...)
	ls := map[string]any{
		"os_crypt": map[string]any{
			"encrypted_key": base64.StdEncoding.EncodeToString(raw),
		},
	}
	data, err := json.Marshal(ls)
	if err != nil {
		t.Fatal(err)
	}
	got, err := EncryptedKeyFromLocalState(data)
	if err != nil {
		t.Fatalf("errore inatteso: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("blob estratto = %x, atteso %x", got, payload)
	}
}

func TestEncryptedKeyFromLocalStateErrori(t *testing.T) {
	casi := map[string]string{
		"json non valido":   `{`,
		"senza chiave":      `{"os_crypt":{}}`,
		"base64 non valido": `{"os_crypt":{"encrypted_key":"@@@"}}`,
		"senza prefisso":    `{"os_crypt":{"encrypted_key":"` + base64.StdEncoding.EncodeToString([]byte("NOPE1234")) + `"}}`,
	}
	for nome, in := range casi {
		if _, err := EncryptedKeyFromLocalState([]byte(in)); err == nil {
			t.Errorf("%s: atteso errore, ottenuto nil", nome)
		}
	}
}

// sigilla cifra plain con AES-256-GCM e restituisce il valore in formato
// Chromium: prefisso | nonce | ciphertext | tag.
func sigilla(t *testing.T, key, plain []byte, prefix string) []byte {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	sealed := gcm.Seal(nil, nonce, plain, nil)
	out := append([]byte(prefix), nonce...)
	return append(out, sealed...)
}

func TestDecryptPassword(t *testing.T) {
	key := make([]byte, 32) // chiave AES-256 a zero, basta per il round-trip
	plain := []byte("segreto-123")
	for _, prefix := range []string{"v10", "v11"} {
		value := sigilla(t, key, plain, prefix)
		got, err := DecryptPassword(key, value)
		if err != nil {
			t.Fatalf("%s: errore inatteso: %v", prefix, err)
		}
		if !bytes.Equal(got, plain) {
			t.Errorf("%s: password = %q, attesa %q", prefix, got, plain)
		}
	}
}

func TestDecryptPasswordLegacy(t *testing.T) {
	key := make([]byte, 32)
	// Valore senza prefisso v10/v11: deve segnalare il formato legacy.
	if _, err := DecryptPassword(key, []byte{0x01, 0x00, 0x00, 0x00, 0xDE, 0xAD}); !errors.Is(err, ErrLegacyDPAPIValue) {
		t.Errorf("atteso ErrLegacyDPAPIValue, ottenuto %v", err)
	}
}

func TestDecryptPasswordManomesso(t *testing.T) {
	key := make([]byte, 32)
	value := sigilla(t, key, []byte("x"), "v10")
	value[len(value)-1] ^= 0xFF // altera il tag: la verifica GCM deve fallire
	if _, err := DecryptPassword(key, value); err == nil {
		t.Error("atteso errore su valore manomesso, ottenuto nil")
	}
}
