@echo off
rem SPDX-License-Identifier: GPL-3.0-or-later
rem
rem Configura Windows per i test di Support Tools. Viene eseguito al primo
rem accesso dell'utente di test (FirstLogonCommands di autounattend.xml);
rem %1 e' l'unita' del CD con i file di configurazione. Al termine spegne la
rem macchina: "go run ./vm install" considera conclusa l'installazione quando
rem QEMU termina.

set SRC=%1
set LOG=C:\stagent\setup.log

mkdir C:\stagent 2>nul
echo Configurazione avviata > %LOG%

rem Agente che esegue i comandi inviati dall'host.
copy /y %SRC%\agent.exe C:\stagent\agent.exe >> %LOG% 2>&1

rem Avvio dell'agente all'accesso dell'utente di test, con i privilegi piu'
rem elevati (senza richiesta UAC) e nella sessione interattiva (/it), cosi'
rem i programmi grafici avviati sono visibili sullo schermo.
schtasks /create /tn "SupportToolsAgent" /tr "C:\stagent\agent.exe" /sc onlogon /ru "{{.User}}" /rp "{{.Password}}" /rl highest /it /f >> %LOG% 2>&1

rem Accesso automatico permanente: AutoLogon di autounattend.xml imposta un
rem numero limitato di accessi (AutoLogonCount), che si elimina.
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v AutoAdminLogon /t REG_SZ /d 1 /f >> %LOG% 2>&1
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v DefaultUserName /t REG_SZ /d "{{.User}}" /f >> %LOG% 2>&1
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v DefaultPassword /t REG_SZ /d "{{.Password}}" /f >> %LOG% 2>&1
reg delete "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v AutoLogonCount /f >> %LOG% 2>&1

rem Niente sospensione, ibernazione e spegnimento dello schermo.
powercfg -change -monitor-timeout-ac 0 >> %LOG% 2>&1
powercfg -change -standby-timeout-ac 0 >> %LOG% 2>&1
powercfg -h off >> %LOG% 2>&1

rem Windows Update rallenterebbe i test e ne cambierebbe le condizioni.
sc config wuauserv start= disabled >> %LOG% 2>&1
sc stop wuauserv >> %LOG% 2>&1

rem Su Windows 10 Defender puo' segnalare per errore gli eseguibili Go non
rem firmati: si esclude la cartella dell'agente (su Windows 7 il comando
rem fallisce senza conseguenze).
powershell -NoProfile -Command "Add-MpPreference -ExclusionPath 'C:\stagent'" >> %LOG% 2>&1

echo Configurazione completata >> %LOG%
shutdown /s /t 5
