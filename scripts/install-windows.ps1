[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Root
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 3.0

function Unquote-Path([object]$Value) {
    if (-not $Value) { return $null }
    $Trimmed = ([string]$Value).Trim()
    if ($Trimmed.StartsWith('"')) {
        $End = $Trimmed.IndexOf('"', 1)
        if ($End -gt 1) { return $Trimmed.Substring(1, $End - 1) }
    }
    return $Trimmed.Split(' ')[0].Trim('"')
}

function Is-Inside([string]$Path, [string]$RootPath) {
    if ((-not $Path) -or (-not $RootPath)) { return $false }
    $FullPath = [IO.Path]::GetFullPath($Path).TrimEnd('\', '/')
    $FullRoot = [IO.Path]::GetFullPath($RootPath).TrimEnd('\', '/')
    if ($FullPath.Equals($FullRoot, [StringComparison]::OrdinalIgnoreCase)) { return $true }
    return $FullPath.StartsWith($FullRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)
}

function Get-RunValue([string]$RunPath) {
    $Item = Get-ItemProperty -Path $RunPath -Name 'Lyn' -ErrorAction SilentlyContinue
    if (-not $Item) { return $null }
    $Property = $Item.PSObject.Properties['Lyn']
    if (-not $Property) { return $null }
    return $Property.Value
}

function Get-WindowsInstallTarget([string]$RepoRoot) {
    $Default = Join-Path ${env:ProgramFiles} 'Lyn\lyn.exe'
    $RunPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
    $Candidate = Unquote-Path (Get-RunValue $RunPath)
    if ($Candidate -and (-not (Is-Inside $Candidate $RepoRoot))) { return $Candidate }
    $ShortcutPath = Join-Path ([Environment]::GetFolderPath('StartMenu')) 'Programs\Lyn.lnk'
    if (Test-Path -LiteralPath $ShortcutPath) {
        $Shell = New-Object -ComObject WScript.Shell
        $Shortcut = $Shell.CreateShortcut($ShortcutPath)
        if ($Shortcut.TargetPath -and (-not (Is-Inside $Shortcut.TargetPath $RepoRoot))) { return $Shortcut.TargetPath }
    }
    return $Default
}

function Quote-PowerShellSingle([string]$Value) {
    return "'" + $Value.Replace("'", "''") + "'"
}

function Install-SignedBroker {
    param(
        [string]$Root,
        [string]$LynSource,
        [string]$HookSource,
        [string]$LynTarget,
        [string]$HookTarget,
        [string]$IconSource,
        [string]$IconTarget
    )
    $DevSign = Join-Path $Root 'scripts\dev-sign.ps1'
    $AsInvoker = Join-Path $Root 'build\windows\wails.exe.asinvoker.manifest'
    $UiAccess = Join-Path $Root 'build\windows\wails.exe.uiaccess.manifest'
    foreach ($Required in @($DevSign, $AsInvoker, $UiAccess, $LynSource, $HookSource)) {
        if (-not (Test-Path -LiteralPath $Required)) { throw "Required file not found at $Required" }
    }
    $TargetDir = Split-Path -Parent $LynTarget
    $LogPath = Join-Path ([IO.Path]::GetTempPath()) ("lyn-install-{0}.log" -f ([Guid]::NewGuid().ToString('N')))
    $Command = @"
& {
    `$ErrorActionPreference = 'Stop'
    Set-StrictMode -Version 3.0
    try {
        `$DevSign = $(Quote-PowerShellSingle $DevSign)
        `$LynSource = $(Quote-PowerShellSingle $LynSource)
        `$HookSource = $(Quote-PowerShellSingle $HookSource)
        `$LynTarget = $(Quote-PowerShellSingle $LynTarget)
        `$HookTarget = $(Quote-PowerShellSingle $HookTarget)
        `$IconSource = $(Quote-PowerShellSingle $IconSource)
        `$IconTarget = $(Quote-PowerShellSingle $IconTarget)
        `$TargetDir = $(Quote-PowerShellSingle $TargetDir)
        `$AsInvoker = $(Quote-PowerShellSingle $AsInvoker)
        `$UiAccess = $(Quote-PowerShellSingle $UiAccess)
        `$LogPath = $(Quote-PowerShellSingle $LogPath)
        & `$DevSign -Command uiaccess -Path `$LynSource -Manifest `$AsInvoker
        & `$DevSign -Command uiaccess -Path `$HookSource -Manifest `$UiAccess
        New-Item -ItemType Directory -Path `$TargetDir -Force | Out-Null
        foreach (`$Stop in @(`$LynTarget, `$HookTarget)) {
            `$StopFull = [IO.Path]::GetFullPath(`$Stop)
            `$Matches = @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
                try { `$_.Path -and [IO.Path]::GetFullPath(`$_.Path).Equals(`$StopFull, [StringComparison]::OrdinalIgnoreCase) } catch { `$false }
            })
            foreach (`$Process in `$Matches) { try { [void]`$Process.CloseMainWindow() } catch { } }
            if (`$Matches.Count -gt 0) { Start-Sleep -Milliseconds 1200 }
            foreach (`$Process in `$Matches) {
                try { `$Process.Refresh(); if (-not `$Process.HasExited) { Stop-Process -Id `$Process.Id -Force -ErrorAction Stop } } catch { }
            }
        }
        Copy-Item -LiteralPath `$LynSource -Destination `$LynTarget -Force
        Copy-Item -LiteralPath `$HookSource -Destination `$HookTarget -Force
        if (Test-Path -LiteralPath `$IconSource) { Copy-Item -LiteralPath `$IconSource -Destination `$IconTarget -Force }
        'OK' | Set-Content -LiteralPath `$LogPath -Encoding UTF8
    } catch {
        `$Detail = `$_.Exception.Message
        if (`$_.ScriptStackTrace) { `$Detail = `$Detail + [Environment]::NewLine + `$_.ScriptStackTrace }
        `$Detail | Set-Content -LiteralPath `$LogPath -Encoding UTF8
        exit 1
    }
}
"@
    $Pwsh = (Get-Command pwsh -ErrorAction SilentlyContinue).Source
    if (-not $Pwsh) { $Pwsh = (Get-Process -Id $PID).Path }
    $Process = Start-Process -FilePath $Pwsh -ArgumentList @('-NoLogo', '-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command', $Command) -Verb RunAs -Wait -PassThru
    if ($Process.ExitCode -ne 0) {
        $Detail = if (Test-Path -LiteralPath $LogPath) { Get-Content -LiteralPath $LogPath -Raw } else { 'No elevated log was written.' }
        throw "Elevated signed install failed with exit code $($Process.ExitCode): $Detail"
    }
    Remove-Item -LiteralPath $LogPath -Force -ErrorAction SilentlyContinue
}

function Set-StartupConfig([string]$ConfigPath) {
    $Config = @{}
    if (Test-Path -LiteralPath $ConfigPath) {
        $Raw = Get-Content -LiteralPath $ConfigPath -Raw
        if ($Raw.Trim()) { $Config = $Raw | ConvertFrom-Json -AsHashtable }
    }
    if ((-not $Config.ContainsKey('startup')) -or (-not ($Config['startup'] -is [System.Collections.IDictionary]))) { $Config['startup'] = @{} }
    $Config['startup']['enabled'] = $true
    $Config['startup']['startHidden'] = $true
    New-Item -ItemType Directory -Path (Split-Path -Parent $ConfigPath) -Force | Out-Null
    $Config | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $ConfigPath -Encoding UTF8
}

$Source = Join-Path $Root 'build\bin\lyn.exe'
$IconSource = Join-Path $Root 'build\windows\icon.ico'
if (-not (Test-Path -LiteralPath $Source)) { throw "Built executable not found at $Source" }
$HookSource = Join-Path $Root 'build\bin\lyn-hook.exe'
Copy-Item -LiteralPath $Source -Destination $HookSource -Force
$Target = Get-WindowsInstallTarget $Root
$TargetDir = Split-Path -Parent $Target
if (-not $TargetDir) {
    $Target = Join-Path ${env:ProgramFiles} 'Lyn\lyn.exe'
    $TargetDir = Split-Path -Parent $Target
}
if (-not (Is-Inside $TargetDir ${env:ProgramFiles})) {
    Write-Warning "Installing to $TargetDir, which is not under Program Files. The uiAccess hotkey broker only gets uiAccess from a secure location, so Win+D will not toggle over elevated windows from here."
}
$HookTarget = Join-Path $TargetDir 'lyn-hook.exe'
$IconTarget = Join-Path $TargetDir 'lyn.ico'
Install-SignedBroker -Root $Root -LynSource $Source -HookSource $HookSource -LynTarget $Target -HookTarget $HookTarget -IconSource $IconSource -IconTarget $IconTarget
$ShortcutIcon = if (Test-Path -LiteralPath $IconSource) { $IconTarget } else { $Target }
$ShortcutPath = Join-Path ([Environment]::GetFolderPath('StartMenu')) 'Programs\Lyn.lnk'
New-Item -ItemType Directory -Path (Split-Path -Parent $ShortcutPath) -Force | Out-Null
$Shell = New-Object -ComObject WScript.Shell
$Shortcut = $Shell.CreateShortcut($ShortcutPath)
$Shortcut.TargetPath = $Target
$Shortcut.WorkingDirectory = $TargetDir
$Shortcut.IconLocation = $ShortcutIcon
$Shortcut.Save()
$RunPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$MachineRun = Get-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'Lyn' -ErrorAction SilentlyContinue
if ($MachineRun) {
    Remove-ItemProperty -Path $RunPath -Name 'Lyn' -ErrorAction SilentlyContinue
} else {
    New-Item -Path $RunPath -Force | Out-Null
    Set-ItemProperty -Path $RunPath -Name 'Lyn' -Value ('"' + $Target + '" --start-hidden')
}
$ConfigPath = Join-Path ([Environment]::GetFolderPath('ApplicationData')) 'lyn\lyn.json'
Set-StartupConfig $ConfigPath
Start-Process -FilePath $Target -ArgumentList @('--start-hidden') -WorkingDirectory $TargetDir -WindowStyle Hidden
Write-Host "Installed Lyn to $Target"
Write-Host 'Signed lyn.exe (asinvoker) and lyn-hook.exe (uiAccess) and deployed the hotkey broker'
Write-Host 'Enabled Lyn startup with hidden background launch'
Write-Host 'Started Lyn in the background'
