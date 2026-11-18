// SPDX-License-Identifier: GPL-3.0-or-later

package users

import "testing"

func TestValidateUserName(t *testing.T) {
	valid := []string{"mario", "Mario Rossi", "ufficio.2", "àèìòù", "a", "12345678901234567890"}
	for _, name := range valid {
		if err := ValidateUserName(name); err != nil {
			t.Errorf("ValidateUserName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{
		"", "   ", " mario", "mario ", "...", ". .",
		"123456789012345678901",
		"mario/rossi", `a\b`, "a:b", "a@b", "a*b", "a?b", "a<b", "a|b", "a\tb",
	}
	for _, name := range invalid {
		if err := ValidateUserName(name); err == nil {
			t.Errorf("ValidateUserName(%q) = nil, want errore", name)
		}
	}
}

func TestValidatePasswords(t *testing.T) {
	if err := ValidatePasswords("abc", "abc"); err != nil {
		t.Errorf("password uguali: %v", err)
	}
	if err := ValidatePasswords("", ""); err != nil {
		t.Errorf("password vuote: %v", err)
	}
	if err := ValidatePasswords("abc", "abd"); err == nil {
		t.Error("password diverse accettate")
	}
}
