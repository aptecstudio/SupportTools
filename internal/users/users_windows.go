// SPDX-License-Identifier: GPL-3.0-or-later

package users

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/sys/windows"

	"github.com/aptecstudio/supporttools/internal/winapi"
)

// User è un account locale.
type User struct {
	Name     string
	FullName string
	Comment  string
	// SID è l'identificativo di sicurezza in forma testuale (S-1-5-21-…);
	// vuoto se non è stato possibile risolverlo.
	SID string
	// Admin indica l'appartenenza al gruppo Administrators.
	Admin bool
	// Disabled indica che l'account è disattivato.
	Disabled bool
	// Current indica l'utente con cui è in esecuzione il programma.
	Current bool
}

// NewUser contiene i dati per creare un account locale.
type NewUser struct {
	Name     string
	FullName string
	Password string
	Admin    bool
	// PasswordNeverExpires imposta "La password non scade mai".
	PasswordNeverExpires bool
	// MustChangePassword obbliga a cambiare la password al primo accesso.
	// Non è compatibile con PasswordNeverExpires.
	MustChangePassword bool
}

// List restituisce tutti gli account locali, ordinati per nome.
func List() ([]User, error) {
	infos, err := winapi.EnumUsers()
	if err != nil {
		return nil, describe("lettura degli utenti", err)
	}
	admins, err := adminSIDs()
	if err != nil {
		return nil, err
	}
	current, _ := currentUserSID()

	list := make([]User, 0, len(infos))
	for _, info := range infos {
		u := User{
			Name:     info.Name,
			FullName: info.FullName,
			Comment:  info.Comment,
			Disabled: info.Flags&winapi.UFAccountDisable != 0,
		}
		if sid, err := userSID(info.Name); err == nil {
			u.SID = sid
			u.Admin = admins[sid]
			u.Current = sid == current
		}
		list = append(list, u)
	}
	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	return list, nil
}

// Create crea un account locale. Se un passaggio successivo alla creazione
// fallisce (nome completo, gruppo, opzioni), l'account viene eliminato per
// non lasciarlo a metà.
func Create(nu NewUser) (err error) {
	if err := ValidateUserName(nu.Name); err != nil {
		return err
	}
	if nu.PasswordNeverExpires && nu.MustChangePassword {
		return errors.New("\"La password non scade mai\" e \"Cambia la password al primo accesso\" " +
			"non possono essere attive insieme")
	}

	var flags uint32
	if nu.PasswordNeverExpires {
		flags |= winapi.UFDontExpirePasswd
	}
	if err := winapi.AddUser(nu.Name, nu.Password, flags); err != nil {
		return describe("creazione dell'utente", err)
	}
	defer func() {
		if err != nil {
			_ = winapi.DeleteUser(nu.Name)
		}
	}()

	if nu.FullName != "" {
		if err := winapi.SetFullName(nu.Name, nu.FullName); err != nil {
			return describe("impostazione del nome completo", err)
		}
	}
	if nu.MustChangePassword {
		if err := winapi.SetPasswordExpired(nu.Name, true); err != nil {
			return describe("impostazione del cambio password al primo accesso", err)
		}
	}

	// Windows di norma aggiunge già i nuovi account al gruppo Users: si
	// ignora l'errore "già membro".
	if err := addToBuiltinGroup(windows.WinBuiltinUsersSid, nu.Name); err != nil {
		return err
	}
	if nu.Admin {
		if err := addToBuiltinGroup(windows.WinBuiltinAdministratorsSid, nu.Name); err != nil {
			return err
		}
	}
	return nil
}

// SetPassword imposta la password dell'utente name.
func SetPassword(name, password string) error {
	if err := winapi.SetPassword(name, password); err != nil {
		return describe("impostazione della password", err)
	}
	return nil
}

// RemovePassword toglie la password dell'utente name. Fallisce se i criteri
// di sicurezza del computer richiedono una lunghezza minima.
func RemovePassword(name string) error {
	if err := winapi.SetPassword(name, ""); err != nil {
		return describe("rimozione della password", err)
	}
	return nil
}

// addToBuiltinGroup aggiunge l'utente al gruppo predefinito indicato.
func addToBuiltinGroup(group windows.WELL_KNOWN_SID_TYPE, name string) error {
	groupName, err := builtinGroupName(group)
	if err != nil {
		return err
	}
	err = winapi.AddLocalGroupMember(groupName, name)
	var ne *winapi.NetError
	if errors.As(err, &ne) && ne.Code == uint32(windows.ERROR_MEMBER_IN_ALIAS) {
		return nil
	}
	if err != nil {
		return describe("aggiunta al gruppo "+groupName, err)
	}
	return nil
}

// builtinGroupName restituisce il nome locale di un gruppo predefinito, che
// dipende dalla lingua di Windows (es. Administrators, Amministratori,
// Administratoren).
func builtinGroupName(group windows.WELL_KNOWN_SID_TYPE) (string, error) {
	sid, err := windows.CreateWellKnownSid(group)
	if err != nil {
		return "", fmt.Errorf("lettura del gruppo predefinito: %w", err)
	}
	name, _, _, err := sid.LookupAccount("")
	if err != nil {
		return "", fmt.Errorf("lettura del gruppo predefinito %s: %w", sid, err)
	}
	return name, nil
}

// adminSIDs restituisce l'insieme dei SID membri del gruppo Administrators.
func adminSIDs() (map[string]bool, error) {
	group, err := builtinGroupName(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	sids, err := winapi.LocalGroupMemberSIDs(group)
	if err != nil {
		return nil, describe("lettura degli amministratori", err)
	}
	set := make(map[string]bool, len(sids))
	for _, s := range sids {
		set[s] = true
	}
	return set, nil
}

// userSID restituisce il SID dell'account locale name. Il nome viene
// qualificato con il nome del computer per non confonderlo con un account
// di dominio omonimo.
func userSID(name string) (string, error) {
	computer, err := windows.ComputerName()
	if err != nil {
		return "", err
	}
	sid, _, _, err := windows.LookupSID("", computer+`\`+name)
	if err != nil {
		return "", err
	}
	return sid.String(), nil
}

// currentUserSID restituisce il SID dell'utente che esegue il programma.
func currentUserSID() (string, error) {
	tu, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	return tu.User.Sid.String(), nil
}

// describe trasforma gli errori di sistema più comuni in messaggi chiari.
func describe(op string, err error) error {
	var ne *winapi.NetError
	if errors.As(err, &ne) {
		switch ne.Code {
		case winapi.NerrUserExists:
			return fmt.Errorf("%s: esiste già un utente con questo nome", op)
		case winapi.NerrGroupExists:
			return fmt.Errorf("%s: esiste già un gruppo con questo nome", op)
		case winapi.NerrUserNotFound:
			return fmt.Errorf("%s: l'utente non esiste più (aggiorna l'elenco)", op)
		case winapi.NerrBadUsername:
			return fmt.Errorf("%s: nome utente non valido", op)
		case winapi.NerrPasswordTooShort:
			return fmt.Errorf("%s: la password non rispetta i criteri di sicurezza del "+
				"computer (lunghezza minima, complessità o cronologia delle password)", op)
		case winapi.NerrLastAdmin:
			return fmt.Errorf("%s: è l'ultimo amministratore del computer", op)
		case uint32(windows.ERROR_ACCESS_DENIED):
			return fmt.Errorf("%s: accesso negato (servono i privilegi di amministratore)", op)
		}
		return fmt.Errorf("%s: %v", op, windows.Errno(ne.Code))
	}
	return fmt.Errorf("%s: %w", op, err)
}
