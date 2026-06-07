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
    $LocalAppData = [Environment]::GetEnvironmentVariable('LOCALAPPDATA')
    if (-not $LocalAppData) { throw 'LOCALAPPDATA is not set' }
    $Default = Join-Path $LocalAppData 'Programs\Lyn\lyn.exe'
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

function Stop-WindowsTarget([string]$Target) {
    $TargetFull = [IO.Path]::GetFullPath($Target)
    $Matches = @(Get-Process | Where-Object {
        try { $_.Path -and [IO.Path]::GetFullPath($_.Path).Equals($TargetFull, [StringComparison]::OrdinalIgnoreCase) } catch { $false }
    })
    foreach ($Process in $Matches) {
        try { [void]$Process.CloseMainWindow() } catch { }
    }
    if ($Matches.Count -gt 0) {
        Start-Sleep -Milliseconds 1200
    }
    foreach ($Process in $Matches) {
        try {
            $Process.Refresh()
            if (-not $Process.HasExited) { Stop-Process -Id $Process.Id -Force }
        } catch {
        }
    }
}

function Quote-PowerShellSingle([string]$Value) {
    return "'" + $Value.Replace("'", "''") + "'"
}

function Copy-WithElevation([string]$Source, [string]$Target) {
    $TargetDir = Split-Path -Parent $Target
    try {
        Copy-Item -LiteralPath $Source -Destination $Target -Force -ErrorAction Stop
        return
    } catch {
        Write-Host "Requesting elevation to replace '$Target'..."
    }

    $LogPath = Join-Path ([IO.Path]::GetTempPath()) ("lyn-install-elevated-{0}.log" -f ([Guid]::NewGuid().ToString('N')))
    $Command = @"
& {
    `$ErrorActionPreference = 'Stop'
    Set-StrictMode -Version 3.0
    try {
        `$Source = $(Quote-PowerShellSingle $Source)
        `$Target = $(Quote-PowerShellSingle $Target)
        `$TargetDir = $(Quote-PowerShellSingle $TargetDir)
        `$LogPath = $(Quote-PowerShellSingle $LogPath)
        New-Item -ItemType Directory -Path `$TargetDir -Force | Out-Null
        `$TargetFull = [IO.Path]::GetFullPath(`$Target)
        `$Matches = @(Get-Process | Where-Object {
            try { `$_.Path -and [IO.Path]::GetFullPath(`$_.Path).Equals(`$TargetFull, [StringComparison]::OrdinalIgnoreCase) } catch { `$false }
        })
        foreach (`$Process in `$Matches) {
            try { [void]`$Process.CloseMainWindow() } catch { }
        }
        if (`$Matches.Count -gt 0) { Start-Sleep -Milliseconds 1200 }
        foreach (`$Process in `$Matches) {
            try {
                `$Process.Refresh()
                if (-not `$Process.HasExited) { Stop-Process -Id `$Process.Id -Force -ErrorAction Stop }
            } catch { }
        }
        Copy-Item -LiteralPath `$Source -Destination `$Target -Force -ErrorAction Stop
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
        throw "Elevated copy failed with exit code $($Process.ExitCode): $Detail"
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
$Target = Get-WindowsInstallTarget $Root
$TargetDir = Split-Path -Parent $Target
if (-not $TargetDir) {
    $Target = Join-Path ([Environment]::GetEnvironmentVariable('LOCALAPPDATA')) 'Programs\Lyn\lyn.exe'
    $TargetDir = Split-Path -Parent $Target
}
try {
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
} catch {
}
Stop-WindowsTarget $Target
Copy-WithElevation $Source $Target
$ShortcutIcon = $Target
if (Test-Path -LiteralPath $IconSource) {
    $IconTarget = Join-Path $TargetDir 'lyn.ico'
    Copy-WithElevation $IconSource $IconTarget
    $ShortcutIcon = $IconTarget
}
$ShortcutPath = Join-Path ([Environment]::GetFolderPath('StartMenu')) 'Programs\Lyn.lnk'
New-Item -ItemType Directory -Path (Split-Path -Parent $ShortcutPath) -Force | Out-Null
$Shell = New-Object -ComObject WScript.Shell
$Shortcut = $Shell.CreateShortcut($ShortcutPath)
$Shortcut.TargetPath = $Target
$Shortcut.WorkingDirectory = $TargetDir
$Shortcut.IconLocation = $ShortcutIcon
$Shortcut.Save()
$RunPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
New-Item -Path $RunPath -Force | Out-Null
Set-ItemProperty -Path $RunPath -Name 'Lyn' -Value ('"' + $Target + '" --start-hidden')
$ConfigPath = Join-Path ([Environment]::GetFolderPath('ApplicationData')) 'lyn\lyn.json'
Set-StartupConfig $ConfigPath
Start-Process -FilePath $Target -ArgumentList @('--start-hidden') -WorkingDirectory $TargetDir -WindowStyle Hidden
Write-Host "Installed Lyn to $Target"
Write-Host 'Enabled Lyn startup with hidden background launch'
Write-Host 'Started Lyn in the background'
