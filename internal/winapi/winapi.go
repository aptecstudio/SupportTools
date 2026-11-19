// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Package winapi dichiara le funzioni di sistema di Windows usate da Support
// Tools che non sono disponibili in golang.org/x/sys/windows: gestione
// degli account locali (netapi32.dll) e dei profili utente (userenv.dll).
//
// Le funzioni sono un sottile strato sopra le API originali: la logica e i
// messaggi per l'utente stanno nei package che le usano.
package winapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	netapi32 = windows.NewLazySystemDLL("netapi32.dll")
	userenv  = windows.NewLazySystemDLL("userenv.dll")

	procNetUserEnum             = netapi32.NewProc("NetUserEnum")
	procNetUserAdd              = netapi32.NewProc("NetUserAdd")
	procNetUserDel              = netapi32.NewProc("NetUserDel")
	procNetUserGetInfo          = netapi32.NewProc("NetUserGetInfo")
	procNetUserSetInfo          = netapi32.NewProc("NetUserSetInfo")
	procNetLocalGroupAddMembers = netapi32.NewProc("NetLocalGroupAddMembers")
	procNetLocalGroupGetMembers = netapi32.NewProc("NetLocalGroupGetMembers")
	procNetApiBufferFree        = netapi32.NewProc("NetApiBufferFree")
	procDeleteProfileW          = userenv.NewProc("DeleteProfileW")
	procGetProfilesDirectoryW   = userenv.NewProc("GetProfilesDirectoryW")
)

const (
	errMoreData         = uintptr(windows.ERROR_MORE_DATA)
	maxPreferredLength  = ^uint32(0) // MAX_PREFERRED_LENGTH
	filterNormalAccount = 0x0002     // FILTER_NORMAL_ACCOUNT
	userPrivUser        = 1          // USER_PRIV_USER
)

// Flag degli account (campo flags di USER_INFO_*).
const (
	UFScript           = 0x0001
	UFAccountDisable   = 0x0002
	UFNormalAccount    = 0x0200
	UFDontExpirePasswd = 0x10000
)

// Codici di errore NetAPI (lmerr.h) usati per i messaggi all'utente.
const (
	NerrBadUsername      = 2202
	NerrUserNotFound     = 2221
	NerrGroupExists      = 2223
	NerrUserExists       = 2224
	NerrPasswordTooShort = 2245
	NerrLastAdmin        = 2452
)

// NetError è un errore restituito da una funzione NetAPI.
type NetError struct {
	Func string
	Code uint32
}

func (e *NetError) Error() string {
	return fmt.Sprintf("%s: %v", e.Func, windows.Errno(e.Code))
}

func netErr(fn string, r uintptr) error {
	if r == 0 {
		return nil
	}
	return &NetError{Func: fn, Code: uint32(r)}
}

// UserInfo20 corrisponde a USER_INFO_20.
type UserInfo20 struct {
	Name     *uint16
	FullName *uint16
	Comment  *uint16
	Flags    uint32
	UserID   uint32
}

// UserInfo1 corrisponde a USER_INFO_1.
type UserInfo1 struct {
	Name        *uint16
	Password    *uint16
	PasswordAge uint32
	Priv        uint32
	HomeDir     *uint16
	Comment     *uint16
	Flags       uint32
	ScriptPath  *uint16
}

// UserInfo3 corrisponde a USER_INFO_3.
type UserInfo3 struct {
	Name            *uint16
	Password        *uint16
	PasswordAge     uint32
	Priv            uint32
	HomeDir         *uint16
	Comment         *uint16
	Flags           uint32
	ScriptPath      *uint16
	AuthFlags       uint32
	FullName        *uint16
	UsrComment      *uint16
	Parms           *uint16
	Workstations    *uint16
	LastLogon       uint32
	LastLogoff      uint32
	AcctExpires     uint32
	MaxStorage      uint32
	UnitsPerWeek    uint32
	LogonHours      *byte
	BadPwCount      uint32
	NumLogons       uint32
	LogonServer     *uint16
	CountryCode     uint32
	CodePage        uint32
	UserID          uint32
	PrimaryGroupID  uint32
	Profile         *uint16
	HomeDirDrive    *uint16
	PasswordExpired uint32
}

// UserInfo1003 corrisponde a USER_INFO_1003 (password).
type UserInfo1003 struct {
	Password *uint16
}

// UserInfo1011 corrisponde a USER_INFO_1011 (nome completo).
type UserInfo1011 struct {
	FullName *uint16
}

// localGroupMembersInfo0 corrisponde a LOCALGROUP_MEMBERS_INFO_0.
type localGroupMembersInfo0 struct {
	SID *windows.SID
}

// localGroupMembersInfo3 corrisponde a LOCALGROUP_MEMBERS_INFO_3.
type localGroupMembersInfo3 struct {
	DomainAndName *uint16
}

// UTF16PtrToString converte una stringa UTF-16 terminata da zero.
func UTF16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	n := 0
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; n++ {
		ptr = unsafe.Add(ptr, unsafe.Sizeof(*p))
	}
	return windows.UTF16ToString(unsafe.Slice(p, n))
}

func utf16Ptr(s string) (*uint16, error) {
	if s == "" {
		return nil, nil
	}
	return windows.UTF16PtrFromString(s)
}

// emptyUTF16 restituisce un puntatore a una stringa vuota (diversa da NULL,
// che per molte API significa "non modificare").
func emptyUTF16() *uint16 {
	return &[]uint16{0}[0]
}

// EnumUsers restituisce gli account locali normali (USER_INFO_20).
func EnumUsers() ([]UserInfo20Value, error) {
	var (
		result []UserInfo20Value
		resume uint32
	)
	for {
		var (
			buf         *byte
			read, total uint32
		)
		r, _, _ := procNetUserEnum.Call(0, 20, uintptr(filterNormalAccount),
			uintptr(unsafe.Pointer(&buf)), uintptr(maxPreferredLength),
			uintptr(unsafe.Pointer(&read)), uintptr(unsafe.Pointer(&total)),
			uintptr(unsafe.Pointer(&resume)))
		if r != 0 && r != errMoreData {
			return nil, netErr("NetUserEnum", r)
		}
		if buf != nil {
			for _, u := range unsafe.Slice((*UserInfo20)(unsafe.Pointer(buf)), read) {
				result = append(result, UserInfo20Value{
					Name:     UTF16PtrToString(u.Name),
					FullName: UTF16PtrToString(u.FullName),
					Comment:  UTF16PtrToString(u.Comment),
					Flags:    u.Flags,
				})
			}
			freeBuffer(buf)
		}
		if r != errMoreData {
			return result, nil
		}
	}
}

// UserInfo20Value è la copia in memoria Go di un USER_INFO_20.
type UserInfo20Value struct {
	Name     string
	FullName string
	Comment  string
	Flags    uint32
}

// AddUser crea un account locale con nome, password e flag indicati.
func AddUser(name, password string, flags uint32) error {
	pName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	pPassword, err := utf16Ptr(password)
	if err != nil {
		return err
	}
	if pPassword == nil {
		pPassword = emptyUTF16()
	}
	info := UserInfo1{
		Name:     pName,
		Password: pPassword,
		Priv:     userPrivUser,
		Flags:    flags | UFScript | UFNormalAccount,
	}
	var parmErr uint32
	r, _, _ := procNetUserAdd.Call(0, 1, uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(&parmErr)))
	return netErr("NetUserAdd", r)
}

// DeleteUser elimina l'account locale name.
func DeleteUser(name string) error {
	pName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	r, _, _ := procNetUserDel.Call(0, uintptr(unsafe.Pointer(pName)))
	return netErr("NetUserDel", r)
}

// setUserInfo chiama NetUserSetInfo con il livello e la struttura indicati.
func setUserInfo(name string, level uint32, info unsafe.Pointer) error {
	pName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	var parmErr uint32
	r, _, _ := procNetUserSetInfo.Call(0, uintptr(unsafe.Pointer(pName)), uintptr(level),
		uintptr(info), uintptr(unsafe.Pointer(&parmErr)))
	return netErr("NetUserSetInfo", r)
}

// SetPassword imposta la password di name; con password vuota la rimuove.
func SetPassword(name, password string) error {
	p, err := utf16Ptr(password)
	if err != nil {
		return err
	}
	if p == nil {
		p = emptyUTF16()
	}
	info := UserInfo1003{Password: p}
	return setUserInfo(name, 1003, unsafe.Pointer(&info))
}

// SetFullName imposta il nome completo di name.
func SetFullName(name, fullName string) error {
	p, err := utf16Ptr(fullName)
	if err != nil {
		return err
	}
	if p == nil {
		p = emptyUTF16()
	}
	info := UserInfo1011{FullName: p}
	return setUserInfo(name, 1011, unsafe.Pointer(&info))
}

// SetPasswordExpired imposta (o toglie) l'obbligo di cambiare la password
// al prossimo accesso.
func SetPasswordExpired(name string, expired bool) error {
	return withUserInfo3(name, func(info *UserInfo3) error {
		info.PasswordExpired = 0
		if expired {
			info.PasswordExpired = 1
		}
		// I campi NULL non vengono modificati: in particolare la password.
		info.Password = nil
		return setUserInfo(name, 3, unsafe.Pointer(info))
	})
}

// UserDetails sono i dettagli di un account letti da USER_INFO_3.
type UserDetails struct {
	FullName string
	Flags    uint32
	// PasswordExpired indica l'obbligo di cambiare la password al prossimo
	// accesso.
	PasswordExpired bool
}

// GetUserDetails legge i dettagli dell'account name.
func GetUserDetails(name string) (UserDetails, error) {
	var d UserDetails
	err := withUserInfo3(name, func(info *UserInfo3) error {
		d = UserDetails{
			FullName:        UTF16PtrToString(info.FullName),
			Flags:           info.Flags,
			PasswordExpired: info.PasswordExpired != 0,
		}
		return nil
	})
	return d, err
}

// withUserInfo3 legge USER_INFO_3 di name e lo passa a fn prima di liberarlo.
func withUserInfo3(name string, fn func(info *UserInfo3) error) error {
	pName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	var buf *byte
	r, _, _ := procNetUserGetInfo.Call(0, uintptr(unsafe.Pointer(pName)), 3,
		uintptr(unsafe.Pointer(&buf)))
	if r != 0 {
		return netErr("NetUserGetInfo", r)
	}
	defer freeBuffer(buf)
	return fn((*UserInfo3)(unsafe.Pointer(buf)))
}

// AddLocalGroupMember aggiunge l'account member al gruppo locale group.
func AddLocalGroupMember(group, member string) error {
	pGroup, err := windows.UTF16PtrFromString(group)
	if err != nil {
		return err
	}
	pMember, err := windows.UTF16PtrFromString(member)
	if err != nil {
		return err
	}
	info := localGroupMembersInfo3{DomainAndName: pMember}
	r, _, _ := procNetLocalGroupAddMembers.Call(0, uintptr(unsafe.Pointer(pGroup)), 3,
		uintptr(unsafe.Pointer(&info)), 1)
	return netErr("NetLocalGroupAddMembers", r)
}

// LocalGroupMemberSIDs restituisce i SID (in forma testuale) dei membri del
// gruppo locale group.
func LocalGroupMemberSIDs(group string) ([]string, error) {
	pGroup, err := windows.UTF16PtrFromString(group)
	if err != nil {
		return nil, err
	}
	var (
		result []string
		resume uintptr
	)
	for {
		var (
			buf         *byte
			read, total uint32
		)
		r, _, _ := procNetLocalGroupGetMembers.Call(0, uintptr(unsafe.Pointer(pGroup)), 0,
			uintptr(unsafe.Pointer(&buf)), uintptr(maxPreferredLength),
			uintptr(unsafe.Pointer(&read)), uintptr(unsafe.Pointer(&total)),
			uintptr(unsafe.Pointer(&resume)))
		if r != 0 && r != errMoreData {
			return nil, netErr("NetLocalGroupGetMembers", r)
		}
		if buf != nil {
			for _, m := range unsafe.Slice((*localGroupMembersInfo0)(unsafe.Pointer(buf)), read) {
				result = append(result, m.SID.String())
			}
			freeBuffer(buf)
		}
		if r != errMoreData {
			return result, nil
		}
	}
}

func freeBuffer(buf *byte) {
	procNetApiBufferFree.Call(uintptr(unsafe.Pointer(buf)))
}

// DeleteProfile elimina il profilo dell'utente con il SID indicato: voci di
// registro e, se possibile, cartella.
func DeleteProfile(sid string) error {
	p, err := windows.UTF16PtrFromString(sid)
	if err != nil {
		return err
	}
	r, _, e := procDeleteProfileW.Call(uintptr(unsafe.Pointer(p)), 0, 0)
	if r == 0 {
		return e
	}
	return nil
}

// ProfilesDirectory restituisce la cartella che contiene i profili utente
// (di solito C:\Users).
func ProfilesDirectory() (string, error) {
	size := uint32(windows.MAX_PATH)
	for {
		buf := make([]uint16, size)
		r, _, e := procGetProfilesDirectoryW.Call(uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)))
		if r != 0 {
			return windows.UTF16ToString(buf), nil
		}
		if e != windows.ERROR_INSUFFICIENT_BUFFER {
			return "", e
		}
	}
}
