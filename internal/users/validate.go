// SPDX-License-Identifier: GPL-3.0-or-later

// Package users gestisce gli account locali di Windows: elenco, creazione,
// cambio e rimozione della password, eliminazione completa con il profilo.
package users

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxUserNameLength è la lunghezza massima di un nome utente locale.
const MaxUserNameLength = 20

// invalidNameChars sono i caratteri non ammessi nei nomi utente di Windows.
const invalidNameChars = `"/\[]:;|=,+*?<>@`

// ValidateUserName controlla che name sia un nome utente locale valido
// secondo le regole di Windows.
func ValidateUserName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("inserisci il nome utente")
	}
	if name != strings.TrimSpace(name) {
		return errors.New("il nome utente non può iniziare o finire con uno spazio")
	}
	if n := utf8.RuneCountInString(name); n > MaxUserNameLength {
		return fmt.Errorf("il nome utente può avere al massimo %d caratteri (ne ha %d)",
			MaxUserNameLength, n)
	}
	if strings.Trim(name, ". ") == "" {
		return errors.New("il nome utente non può essere composto solo da punti e spazi")
	}
	for _, r := range name {
		if strings.ContainsRune(invalidNameChars, r) {
			return fmt.Errorf("il nome utente non può contenere il carattere %q; "+
				"non sono ammessi: %s", r, invalidNameChars)
		}
		if unicode.IsControl(r) {
			return errors.New("il nome utente contiene caratteri non validi")
		}
	}
	return nil
}

// ValidatePasswords controlla che la password e la sua conferma coincidano.
func ValidatePasswords(password, confirm string) error {
	if password != confirm {
		return errors.New("la password e la conferma non coincidono")
	}
	return nil
}
