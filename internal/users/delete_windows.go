// SPDX-License-Identifier: GPL-3.0-or-later

package users

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/aptecstudio/supporttools/internal/winapi"
)

// profileListKey è la chiave di registro con l'elenco dei profili utente.
const profileListKey = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList`

// maxReportedFailures è il numero massimo di file non eliminati elencati
// nel messaggio di errore.
const maxReportedFailures = 5

// Progress riceve l'avanzamento di un'operazione lunga; è soddisfatta da
// *progress.Reporter.
type Progress interface {
	Step(name string)
	SetStatus(text string)
	SetTotal(total int64)
	Add(n int64)
}

// DeletePlan descrive l'eliminazione completa di un utente: va preparata
// con PlanDelete, mostrata all'utente per conferma ed eseguita con Execute.
type DeletePlan struct {
	User User
	// ProfilePath è la cartella del profilo; vuota se l'utente non ha mai
	// effettuato l'accesso.
	ProfilePath string
	// ManualFolderDelete indica che la cartella verrà cancellata file per
	// file con avanzamento. È falso se la cartella non si trova direttamente
	// nella cartella dei profili: in quel caso, per sicurezza, se ne occupa
	// solo Windows (DeleteProfile).
	ManualFolderDelete bool
}

// PlanDelete verifica che l'utente name si possa eliminare e prepara il piano.
func PlanDelete(name string) (*DeletePlan, error) {
	u, err := find(name)
	if err != nil {
		return nil, err
	}
	if err := checkDeletable(u); err != nil {
		return nil, err
	}
	plan := &DeletePlan{User: u}
	path, err := profilePath(u.SID)
	if err != nil {
		return nil, err
	}
	if path != "" {
		plan.ProfilePath = path
		plan.ManualFolderDelete, err = isDirectProfileFolder(path)
		if err != nil {
			return nil, err
		}
	}
	return plan, nil
}

// Execute elimina profilo e account. L'account si elimina per ultimo: se un
// passaggio fallisce, l'utente resta e l'operazione si può ripetere.
func (p *DeletePlan) Execute(ctx context.Context, prog Progress) error {
	name := p.User.Name

	prog.Step("Verifica dell'utente " + name)
	u, err := find(name)
	if err != nil {
		return err
	}
	if u.SID != p.User.SID {
		return errors.New("l'utente è cambiato dopo la conferma: aggiorna l'elenco e riprova")
	}
	if err := checkDeletable(u); err != nil {
		return err
	}

	// leftover descrive i file che la cancellazione manuale non ha potuto
	// eliminare: DeleteProfile può ancora riuscirci, altrimenti finisce
	// nel messaggio di errore.
	leftover := ""
	if p.ManualFolderDelete && dirExists(p.ProfilePath) {
		if leftover, err = deleteProfileFolder(ctx, p.ProfilePath, prog); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	prog.Step("Rimozione del profilo dal registro")
	if err := removeProfile(u.SID, p.ProfilePath); err != nil {
		if leftover != "" {
			return fmt.Errorf("%v\n\n%s", err, leftover)
		}
		return err
	}
	prog.Add(1)

	if err := ctx.Err(); err != nil {
		return err
	}
	prog.Step("Eliminazione dell'account " + name)
	if err := winapi.DeleteUser(name); err != nil {
		return describe("eliminazione dell'account", err)
	}
	prog.Add(1)
	return nil
}

// find restituisce l'account locale name.
func find(name string) (User, error) {
	list, err := List()
	if err != nil {
		return User{}, err
	}
	for _, u := range list {
		if strings.EqualFold(u.Name, name) {
			if u.SID == "" {
				return User{}, fmt.Errorf("impossibile leggere l'identificativo dell'utente %s", name)
			}
			return u, nil
		}
	}
	return User{}, fmt.Errorf("l'utente %s non esiste più: aggiorna l'elenco", name)
}

// checkDeletable impedisce di eliminare l'utente corrente, un utente
// connesso o l'ultimo amministratore attivo.
func checkDeletable(u User) error {
	if u.Current {
		return fmt.Errorf("non puoi eliminare %s: è l'utente con cui stai usando il programma", u.Name)
	}
	if profileLoaded(u.SID) {
		return fmt.Errorf("non puoi eliminare %s: l'utente è connesso o il suo profilo è in uso. "+
			"Disconnettilo (o riavvia il computer) e riprova", u.Name)
	}
	if u.Admin && !u.Disabled {
		list, err := List()
		if err != nil {
			return err
		}
		others := 0
		for _, o := range list {
			if o.Admin && !o.Disabled && o.SID != u.SID {
				others++
			}
		}
		if others == 0 {
			return fmt.Errorf("non puoi eliminare %s: è l'ultimo amministratore attivo del computer", u.Name)
		}
	}
	return nil
}

// profileLoaded indica se il profilo è caricato (utente connesso o profilo
// in uso da un servizio): in quel caso la sua chiave è in HKEY_USERS.
func profileLoaded(sid string) bool {
	k, err := registry.OpenKey(registry.USERS, sid, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// profilePath restituisce la cartella del profilo dal registro, o una
// stringa vuota se l'utente non ha un profilo.
func profilePath(sid string) (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, profileListKey+`\`+sid, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("lettura del profilo dal registro: %w", err)
	}
	defer k.Close()

	v, _, err := k.GetStringValue("ProfileImagePath")
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("lettura della cartella del profilo: %w", err)
	}
	v, err = registry.ExpandString(v)
	if err != nil {
		return "", err
	}
	return filepath.Clean(v), nil
}

// isDirectProfileFolder indica se path è una sottocartella diretta della
// cartella dei profili (es. C:\Users\mario) e non una delle cartelle comuni.
func isDirectProfileFolder(path string) (bool, error) {
	root, err := winapi.ProfilesDirectory()
	if err != nil {
		return false, fmt.Errorf("lettura della cartella dei profili: %w", err)
	}
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Dir(path), filepath.Clean(root)) {
		return false, nil
	}
	switch strings.ToLower(filepath.Base(path)) {
	case "public", "default", "default user", "all users":
		return false, nil
	}
	return true, nil
}

// removeProfile elimina le voci di registro del profilo (ed eventuali resti
// della cartella) tramite DeleteProfile. Se DeleteProfile fallisce ma la
// cartella e la voce di registro risultano già rimosse, non è un errore.
func removeProfile(sid, path string) error {
	errProfile := winapi.DeleteProfile(sid)

	keyLeft := false
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, profileListKey+`\`+sid, registry.QUERY_VALUE); err == nil {
		k.Close()
		keyLeft = true
	}
	if keyLeft {
		if err := registry.DeleteKey(registry.LOCAL_MACHINE, profileListKey+`\`+sid); err != nil {
			if errProfile != nil {
				return fmt.Errorf("rimozione del profilo dal registro: %v", errProfile)
			}
			return fmt.Errorf("rimozione del profilo dal registro: %w", err)
		}
	}
	if path != "" && dirExists(path) {
		if errProfile != nil {
			return fmt.Errorf("la cartella %s non è stata eliminata completamente: %v", path, errProfile)
		}
		return fmt.Errorf("la cartella %s non è stata eliminata completamente", path)
	}
	return nil
}

// deleteProfileFolder cancella la cartella del profilo con avanzamento.
// Giunzioni e collegamenti simbolici vengono rimossi senza seguirli, così
// non si cancella mai nulla al di fuori della cartella. Gli elementi che non
// si possono eliminare non interrompono l'operazione: vengono descritti nel
// testo restituito (vuoto se è stato eliminato tutto).
func deleteProfileFolder(ctx context.Context, root string, prog Progress) (string, error) {
	prog.Step("Analisi della cartella " + root)
	count := int64(0)
	err := walkNoFollow(ctx, root, func(path string, d fs.DirEntry, reparse bool) error {
		count++
		prog.SetStatus(fmt.Sprintf("%d elementi trovati", count))
		return nil
	})
	if err != nil {
		return "", err
	}
	// Due unità in più per la rimozione dal registro e dell'account.
	prog.SetTotal(count + 2)

	prog.Step("Eliminazione della cartella " + root)
	var (
		dirs     []string
		failures []string
		failed   int
	)
	err = walkNoFollow(ctx, root, func(path string, d fs.DirEntry, reparse bool) error {
		defer prog.Add(1)
		rel, _ := filepath.Rel(root, path)
		prog.SetStatus(rel)
		if d.IsDir() && !reparse {
			// Le cartelle si eliminano alla fine, dopo il loro contenuto.
			dirs = append(dirs, path)
			return nil
		}
		if err := removeEntry(path); err != nil {
			failed++
			if len(failures) < maxReportedFailures {
				failures = append(failures, rel)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		// Una cartella che non si elimina contiene file rimasti, già contati.
		_ = removeEntry(dirs[i])
	}

	if failed == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d elementi non sono stati eliminati (file in uso o protetti), tra cui:\n%s",
		failed, strings.Join(failures, "\n")), nil
}

// walkNoFollow visita root in profondità senza entrare in giunzioni e
// collegamenti; fn riceve reparse=true per questi elementi. root stesso non
// viene passato a fn. La visita si interrompe se ctx viene annullato.
func walkNoFollow(ctx context.Context, root string, fn func(path string, d fs.DirEntry, reparse bool) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			if path == root {
				return err
			}
			// Cartella non leggibile: WalkDir l'ha già passata a fn prima di
			// leggerla, quindi è già contata (e la sua eliminazione fallirà e
			// verrà segnalata); si salta il contenuto.
			return filepath.SkipDir
		}
		if path == root {
			return nil
		}
		reparse := isReparsePoint(d)
		if err := fn(path, d, reparse); err != nil {
			return err
		}
		if reparse && d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
}

// isReparsePoint indica se l'elemento è una giunzione, un collegamento
// simbolico o un altro reparse point.
func isReparsePoint(d fs.DirEntry) bool {
	if d.Type()&fs.ModeSymlink != 0 {
		return true
	}
	info, err := d.Info()
	if err != nil {
		return false
	}
	if attr, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return attr.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
	}
	return false
}

// removeEntry elimina un file, una cartella vuota o un collegamento. Se il
// primo tentativo fallisce toglie gli attributi (sola lettura, sistema) e
// riprova: le cartelle speciali del profilo sono spesso in sola lettura.
func removeEntry(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	p, perr := windows.UTF16PtrFromString(longPath(path))
	if perr != nil {
		return err
	}
	if windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL) != nil {
		return err
	}
	return os.Remove(path)
}

// longPath aggiunge il prefisso \\?\ ai percorsi assoluti, per superare il
// limite di 260 caratteri nelle API usate direttamente.
func longPath(path string) string {
	if strings.HasPrefix(path, `\\`) || !filepath.IsAbs(path) {
		return path
	}
	return `\\?\` + path
}

func dirExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}
