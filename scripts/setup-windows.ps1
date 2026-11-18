# SPDX-License-Identifier: GPL-3.0-or-later

<#
.SYNOPSIS
Installa le dipendenze di sviluppo di Support Tools su Windows.

.DESCRIPTION
Installa:
  - la toolchain Go 1.20.14 in %LOCALAPPDATA%\Programs\Go (solo per l'utente
    corrente, non servono privilegi di amministratore);
  - gli strumenti di sviluppo: rsrc (manifest/icona) e goimports;
  - i moduli Go elencati in go.mod.

Lo script è idempotente: può essere rieseguito senza effetti collaterali.
Richiede PowerShell 5.1 o successivo.

.PARAMETER Force
Sostituisce un'eventuale versione di Go diversa già presente nella cartella
di installazione.

.EXAMPLE
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\setup-windows.ps1
#>

[CmdletBinding()]
param(
    [switch]$Force
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# Senza barra di avanzamento Invoke-WebRequest è molto più veloce su PowerShell 5.1.
$ProgressPreference = 'SilentlyContinue'

# Versioni fissate: Go 1.20.x è l'ultima release che supporta Windows 7.
$GoVersion = '1.20.14'
$GoRoot = Join-Path $env:LOCALAPPDATA 'Programs\Go'
$RsrcVersion = 'v0.10.2'
# Ultima versione di x/tools compatibile con Go 1.20.
$GoimportsVersion = 'v0.24.0'

# Checksum ufficiali pubblicati su https://go.dev/dl/
$GoChecksums = @{
    'go1.20.14.windows-amd64.zip' = '0e0d0190406ead891d94ecf00f961bb5cfa15ddd47499d2649f12eee80aee110'
    'go1.20.14.windows-386.zip'   = 'ceaa0e7a7c89604027d67bbcce72b8574080d5c34b68bd3152338402c9253ce2'
    'go1.20.14.windows-arm64.zip' = '67dfb9de6c4775d3deffb3bf87c204f22b2d0ddebebc6eec5917a69e27af7370'
}

function Write-Info([string]$Message) {
    Write-Host '==> ' -ForegroundColor Blue -NoNewline
    Write-Host $Message
}

function Write-Warn([string]$Message) {
    Write-Host "[!] $Message" -ForegroundColor Yellow
}

function Stop-Setup([string]$Message) {
    Write-Host "[x] $Message" -ForegroundColor Red
    exit 1
}

# Invoke-Native esegue un comando esterno e interrompe lo script se fallisce.
function Invoke-Native([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        Stop-Setup "Comando non riuscito: $Command $($Arguments -join ' ')"
    }
}

function Test-Requirements {
    if ($PSVersionTable.PSVersion.Major -lt 5 -or
        ($PSVersionTable.PSVersion.Major -eq 5 -and $PSVersionTable.PSVersion.Minor -lt 1)) {
        Stop-Setup "Serve PowerShell 5.1 o successivo (presente: $($PSVersionTable.PSVersion))."
    }
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Stop-Setup 'Comando mancante: git. Installalo da https://git-scm.com/ e rilancia lo script.'
    }
}

# Get-GoArchive restituisce il nome dell'archivio Go adatto al sistema.
function Get-GoArchive {
    # Con PowerShell a 32 bit su sistema a 64 bit l'architettura reale è in
    # PROCESSOR_ARCHITEW6432.
    $arch = $env:PROCESSOR_ARCHITEW6432
    if (-not $arch) {
        $arch = $env:PROCESSOR_ARCHITECTURE
    }
    switch ($arch) {
        'AMD64' { $goArch = 'amd64' }
        'x86' { $goArch = '386' }
        'ARM64' { $goArch = 'arm64' }
        default { Stop-Setup "Architettura non supportata: $arch" }
    }
    return "go$GoVersion.windows-$goArch.zip"
}

function Install-Go {
    $goExe = Join-Path $GoRoot 'bin\go.exe'
    $current = ''
    if (Test-Path $goExe) {
        $current = (& $goExe env GOVERSION 2>$null | Out-String).Trim()
    }

    if ($current -eq "go$GoVersion") {
        Write-Info "Go $GoVersion già installato in $GoRoot"
        return
    }
    if ($current -and -not $Force) {
        Stop-Setup "In $GoRoot è presente $current. Rilancia con -Force per sostituirlo con Go $GoVersion."
    }

    $file = Get-GoArchive
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        $archive = Join-Path $tmp $file

        Write-Info "Download di $file"
        # TLS 1.2 non è abilitato di default su .NET Framework più vecchi.
        [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor
            [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -UseBasicParsing -Uri "https://go.dev/dl/$file" -OutFile $archive

        Write-Info 'Verifica del checksum'
        $sum = (Get-FileHash -Algorithm SHA256 -Path $archive).Hash.ToLowerInvariant()
        if ($sum -ne $GoChecksums[$file]) {
            Stop-Setup "Checksum non valido per ${file}: download corrotto o manomesso."
        }

        Write-Info "Installazione in $GoRoot"
        if (Test-Path $GoRoot) {
            Remove-Item -Recurse -Force $GoRoot
        }
        # L'archivio contiene una cartella "go": si estrae accanto e si rinomina.
        $parent = Split-Path $GoRoot -Parent
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
        Expand-Archive -Path $archive -DestinationPath $tmp
        Move-Item -Path (Join-Path $tmp 'go') -Destination $GoRoot
    }
    finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }
}

# Send-EnvironmentChange avvisa le applicazioni aperte (Esplora risorse, ecc.)
# che le variabili d'ambiente sono cambiate.
function Send-EnvironmentChange {
    if (-not ('SupportTools.NativeMethods' -as [type])) {
        Add-Type -Namespace SupportTools -Name NativeMethods -MemberDefinition @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam,
    string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
    }
    $HWND_BROADCAST = [IntPtr]0xffff
    $WM_SETTINGCHANGE = 0x1A
    $SMTO_ABORTIFHUNG = 0x2
    $result = [UIntPtr]::Zero
    [SupportTools.NativeMethods]::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE,
        [UIntPtr]::Zero, 'Environment', $SMTO_ABORTIFHUNG, 5000, [ref]$result) | Out-Null
}

# Set-UserPath rende disponibili go e gli strumenti installati, sia per
# questa esecuzione sia per i terminali futuri (PATH dell'utente).
function Set-UserPath {
    $goBin = Join-Path $GoRoot 'bin'
    $env:Path = "$goBin;$env:Path"
    $gopathBin = Join-Path ((& go env GOPATH | Out-String).Trim()) 'bin'
    $env:Path = "$gopathBin;$env:Path"

    # Il PATH dell'utente si legge senza espandere le variabili (es. %USERPROFILE%)
    # e si riscrive come REG_EXPAND_SZ, per non alterare le voci esistenti.
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
    try {
        $raw = [string]$key.GetValue('Path', '',
            [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $entries = @($raw -split ';' | Where-Object { $_ })
        $expanded = @($entries | ForEach-Object {
                [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\')
            })

        $missing = @($goBin, $gopathBin | Where-Object { $expanded -notcontains $_.TrimEnd('\') })
        if ($missing.Count -eq 0) {
            return
        }
        Write-Info "Aggiunta di $($missing -join ' e ') al PATH dell'utente"
        $key.SetValue('Path', (($missing + $entries) -join ';'),
            [Microsoft.Win32.RegistryValueKind]::ExpandString)
    }
    finally {
        $key.Close()
    }
    Send-EnvironmentChange
}

function Test-ConflictingGo {
    # Un altro Go nel PATH di sistema (es. installato con l'MSI in
    # C:\Program Files\Go) avrebbe la precedenza su quello dell'utente.
    $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    foreach ($dir in @($machinePath -split ';' | Where-Object { $_ })) {
        $candidate = Join-Path $dir 'go.exe'
        if (Test-Path $candidate) {
            $version = (& $candidate env GOVERSION 2>$null | Out-String).Trim()
            if ($version -ne "go$GoVersion") {
                Write-Warn "Nel PATH di sistema c'è $version ($candidate), che ha la precedenza su Go $GoVersion. Rimuovilo o toglilo dal PATH di sistema."
            }
        }
    }
}

function Install-Tools {
    # I moduli degli strumenti non devono richiedere una versione di Go più recente.
    $env:GOTOOLCHAIN = 'local'

    Write-Info "Installazione di rsrc $RsrcVersion"
    Invoke-Native 'go' @('install', "github.com/akavel/rsrc@$RsrcVersion")

    Write-Info "Installazione di goimports $GoimportsVersion"
    Invoke-Native 'go' @('install', "golang.org/x/tools/cmd/goimports@$GoimportsVersion")
}

function Get-Modules {
    $root = Split-Path $PSScriptRoot -Parent

    Write-Info 'Download dei moduli Go del progetto'
    Push-Location $root
    try {
        Invoke-Native 'go' @('mod', 'download')
        Invoke-Native 'go' @('mod', 'verify')
    }
    finally {
        Pop-Location
    }
}

Test-Requirements
Install-Go
Set-UserPath
Test-ConflictingGo
Install-Tools
Get-Modules

Write-Info "Completato: $((& go version | Out-String).Trim())"
Write-Host 'Apri un nuovo terminale per avere go nel PATH.'
