// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

// Test di integrazione: creano, modificano ed eliminano account e profili
// reali. Vanno eseguiti SOLO nelle macchine virtuali di test
// (go run ./vm test <macchina>), mai sul proprio computer.

package users

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/aptecstudio/supporttools/internal/winapi"
)

const testPassword = "Prova-12345!"

var (
	advapi32                = windows.NewLazySystemDLL("advapi32.dll")
	userenv                 = windows.NewLazySystemDLL("userenv.dll")
	procLogonUserW          = advapi32.NewProc("LogonUserW")
	procLoadUserProfileW    = userenv.NewProc("LoadUserProfileW")
	procUnloadUserProfile   = userenv.NewProc("UnloadUserProfile")
	errPasswordMustChange   = windows.Errno(1907) // ERROR_PASSWORD_MUST_CHANGE
	logon32LogonInteractive = 2
)

// profileInfo corrisponde a PROFILEINFOW.
type profileInfo struct {
	size        uint32
	flags       uint32
	userName    *uint16
	profilePath *uint16
	defaultPath *uint16
	serverName  *uint16
	policyPath  *uint16
	profile     windows.Handle
}

func TestMain(m *testing.M) {
	// LoadUserProfile richiede i privilegi di backup e ripristino attivi.
	for _, p := range []string{"SeBackupPrivilege", "SeRestorePrivilege"} {
		if err := enablePrivilege(p); err != nil {
			fmt.Fprintf(os.Stderr, "abilitazione di %s: %v\n", p, err)
			os.Exit(1)
		}
	}
	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

// newUserName restituisce un nome utente di prova univoco.
func newUserName(t *testing.T) string {
	t.Helper()
	name := fmt.Sprintf("sttest%06d", rand.Intn(1000000))
	t.Cleanup(func() { cleanupUser(name) })
	return name
}

// cleanupUser elimina, se rimasti, profilo e account di prova.
func cleanupUser(name string) {
	if sid, err := userSID(name); err == nil {
		_ = winapi.DeleteProfile(sid)
	}
	_ = winapi.DeleteUser(name)
}

func createUser(t *testing.T, nu NewUser) {
	t.Helper()
	if err := Create(nu); err != nil {
		t.Fatalf("Create(%s): %v", nu.Name, err)
	}
}

func mustFind(t *testing.T, name string) User {
	t.Helper()
	u, err := find(name)
	if err != nil {
		t.Fatalf("find(%s): %v", name, err)
	}
	return u
}

func logon(name, password string) (windows.Token, error) {
	pName, _ := windows.UTF16PtrFromString(name)
	pDomain, _ := windows.UTF16PtrFromString(".")
	pPassword, _ := windows.UTF16PtrFromString(password)
	var token windows.Token
	r, _, err := procLogonUserW.Call(uintptr(unsafe.Pointer(pName)), uintptr(unsafe.Pointer(pDomain)),
		uintptr(unsafe.Pointer(pPassword)), uintptr(logon32LogonInteractive), 0,
		uintptr(unsafe.Pointer(&token)))
	if r == 0 {
		return 0, err
	}
	return token, nil
}

func checkLogon(t *testing.T, name, password string) {
	t.Helper()
	token, err := logon(name, password)
	if err != nil {
		t.Fatalf("accesso di %s con la password %q non riuscito: %v", name, password, err)
	}
	token.Close()
}

// loadProfile accede come name e carica il suo profilo, creandolo se non
// esiste; la funzione restituita lo scarica.
func loadProfile(t *testing.T, name, password string) (unload func()) {
	t.Helper()
	token, err := logon(name, password)
	if err != nil {
		t.Fatalf("accesso di %s: %v", name, err)
	}
	pName, _ := windows.UTF16PtrFromString(name)
	pi := profileInfo{flags: 1 /* PI_NOUI */, userName: pName}
	pi.size = uint32(unsafe.Sizeof(pi))
	r, _, err := procLoadUserProfileW.Call(uintptr(token), uintptr(unsafe.Pointer(&pi)))
	if r == 0 {
		token.Close()
		t.Fatalf("caricamento del profilo di %s: %v", name, err)
	}
	return func() {
		procUnloadUserProfile.Call(uintptr(token), uintptr(pi.profile))
		token.Close()
	}
}

func enablePrivilege(name string) error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(),
		windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	var luid windows.LUID
	pName, _ := windows.UTF16PtrFromString(name)
	if err := windows.LookupPrivilegeValue(nil, pName, &luid); err != nil {
		return err
	}
	tp := windows.Tokenprivileges{PrivilegeCount: 1}
	tp.Privileges[0] = windows.LUIDAndAttributes{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}
	return windows.AdjustTokenPrivileges(token, false, &tp, 0, nil, nil)
}

// waitUnloaded attende che Windows scarichi del tutto il profilo.
func waitUnloaded(t *testing.T, sid string) {
	t.Helper()
	for i := 0; i < 50 && profileLoaded(sid); i++ {
		time.Sleep(200 * time.Millisecond)
	}
	if profileLoaded(sid) {
		t.Fatal("il profilo risulta ancora caricato")
	}
}

// recorder registra l'avanzamento ricevuto da DeletePlan.Execute.
type recorder struct {
	steps []string
	total int64
	done  int64
}

func (r *recorder) Step(name string)     { r.steps = append(r.steps, name) }
func (r *recorder) SetStatus(string)     {}
func (r *recorder) SetTotal(total int64) { r.total = total }
func (r *recorder) Add(n int64)          { r.done += n }

func TestCreateStandardUser(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, FullName: "Utente di prova", Password: testPassword,
		PasswordNeverExpires: true})

	u := mustFind(t, name)
	if u.FullName != "Utente di prova" || u.Admin || u.Disabled || u.Current {
		t.Errorf("utente creato non corretto: %+v", u)
	}
	d, err := winapi.GetUserDetails(name)
	if err != nil {
		t.Fatal(err)
	}
	if d.Flags&winapi.UFDontExpirePasswd == 0 {
		t.Error("\"La password non scade mai\" non impostata")
	}
	if d.PasswordExpired {
		t.Error("cambio password al primo accesso impostato senza richiesta")
	}
	checkLogon(t, name, testPassword)
}

func TestCreateAdmin(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, Password: testPassword, Admin: true})
	if u := mustFind(t, name); !u.Admin {
		t.Error("l'utente non risulta amministratore")
	}
}

func TestCreateMustChangePassword(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, Password: testPassword, MustChangePassword: true})

	d, err := winapi.GetUserDetails(name)
	if err != nil {
		t.Fatal(err)
	}
	if !d.PasswordExpired {
		t.Error("cambio password al primo accesso non impostato")
	}
	if _, err := logon(name, testPassword); !errors.Is(err, errPasswordMustChange) {
		t.Errorf("accesso: errore %v, atteso %v", err, errPasswordMustChange)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	name := newUserName(t)
	err := Create(NewUser{Name: name, Password: testPassword,
		PasswordNeverExpires: true, MustChangePassword: true})
	if err == nil {
		t.Error("opzioni incompatibili accettate")
	}
	if _, err := find(name); err == nil {
		t.Error("utente creato nonostante le opzioni incompatibili")
	}
	if err := Create(NewUser{Name: "a/b", Password: testPassword}); err == nil {
		t.Error("nome non valido accettato")
	}
}

func TestCreateDuplicate(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, Password: testPassword})
	err := Create(NewUser{Name: name, Password: testPassword})
	if err == nil || !strings.Contains(err.Error(), "esiste già") {
		t.Errorf("seconda creazione: errore %v, atteso \"esiste già\"", err)
	}
}

func TestSetAndRemovePassword(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, Password: testPassword})

	const newPassword = "Nuova-67890!"
	if err := SetPassword(name, newPassword); err != nil {
		t.Fatal(err)
	}
	checkLogon(t, name, newPassword)
	if _, err := logon(name, testPassword); err == nil {
		t.Error("la vecchia password funziona ancora")
	}

	if err := RemovePassword(name); err != nil {
		t.Fatal(err)
	}
	checkLogon(t, name, "")
}

func TestDeleteUserWithoutProfile(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, Password: testPassword})

	plan, err := PlanDelete(name)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ProfilePath != "" {
		t.Errorf("profilo inatteso: %s", plan.ProfilePath)
	}
	if err := plan.Execute(context.Background(), new(recorder)); err != nil {
		t.Fatal(err)
	}
	if _, err := find(name); err == nil {
		t.Error("l'utente esiste ancora")
	}
}

func TestDeleteUserWithProfile(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, Password: testPassword})
	sid := mustFind(t, name).SID

	unload := loadProfile(t, name, testPassword)
	path, err := profilePath(sid)
	if err != nil || path == "" {
		unload()
		t.Fatalf("profilo non creato: %q, %v", path, err)
	}
	unload()
	waitUnloaded(t, sid)

	// Contenuti difficili: percorso oltre 260 caratteri, cartella e file in
	// sola lettura, file nascosto di sistema.
	deep := path
	for i := 0; len(deep) < 300; i++ {
		deep = filepath.Join(deep, fmt.Sprintf("cartella-profonda-%02d", i))
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ro := filepath.Join(path, "SolaLettura")
	if err := os.MkdirAll(ro, 0o755); err != nil {
		t.Fatal(err)
	}
	roFile := filepath.Join(ro, "file.txt")
	if err := os.WriteFile(roFile, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	setAttributes(t, ro, windows.FILE_ATTRIBUTE_READONLY)
	hidden := filepath.Join(path, "nascosto.dat")
	if err := os.WriteFile(hidden, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	setAttributes(t, hidden, windows.FILE_ATTRIBUTE_HIDDEN|windows.FILE_ATTRIBUTE_SYSTEM)

	// Giunzione verso una cartella esterna: il suo contenuto deve restare.
	outside := filepath.Join(os.TempDir(), "sentinella-"+name)
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outside)
	sentinel := filepath.Join(outside, "non-cancellare.txt")
	if err := os.WriteFile(sentinel, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J",
		filepath.Join(path, "Giunzione"), outside).CombinedOutput(); err != nil {
		t.Fatalf("creazione della giunzione: %v\n%s", err, out)
	}

	plan, err := PlanDelete(name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(plan.ProfilePath, path) || !plan.ManualFolderDelete {
		t.Fatalf("piano non corretto: %+v", plan)
	}
	rec := new(recorder)
	if err := plan.Execute(context.Background(), rec); err != nil {
		t.Fatal(err)
	}

	if dirExists(path) {
		t.Errorf("la cartella del profilo esiste ancora: %s", path)
	}
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, profileListKey+`\`+sid, registry.QUERY_VALUE); err == nil {
		k.Close()
		t.Error("la voce ProfileList esiste ancora")
	}
	if _, err := find(name); err == nil {
		t.Error("l'utente esiste ancora")
	}
	if !fileExists(sentinel) {
		t.Error("eliminato un file fuori dal profilo attraverso la giunzione")
	}
	if rec.total == 0 || rec.done != rec.total {
		t.Errorf("avanzamento non coerente: %d su %d", rec.done, rec.total)
	}
	t.Logf("passaggi: %q", rec.steps)
}

func TestDeleteRefusesLoadedProfile(t *testing.T) {
	name := newUserName(t)
	createUser(t, NewUser{Name: name, Password: testPassword})
	unload := loadProfile(t, name, testPassword)
	defer unload()

	_, err := PlanDelete(name)
	if err == nil || !strings.Contains(err.Error(), "connesso") {
		t.Errorf("errore %v, atteso il rifiuto per profilo in uso", err)
	}
}

func TestDeleteRefusesCurrentUser(t *testing.T) {
	sid, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	s, err := windows.StringToSid(sid)
	if err != nil {
		t.Fatal(err)
	}
	name, _, _, err := s.LookupAccount("")
	if err != nil {
		t.Fatal(err)
	}
	_, err = PlanDelete(name)
	if err == nil || !strings.Contains(err.Error(), "stai usando") {
		t.Errorf("errore %v, atteso il rifiuto per l'utente corrente", err)
	}
}

func setAttributes(t *testing.T, path string, attrs uint32) {
	t.Helper()
	p, _ := windows.UTF16PtrFromString(path)
	if err := windows.SetFileAttributes(p, attrs); err != nil {
		t.Fatalf("attributi di %s: %v", path, err)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
