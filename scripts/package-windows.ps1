[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Root,
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [Parameter(Mandatory = $true)]
    [string]$Output
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Source = Join-Path $Root 'build\bin\lyn.exe'
$Icon = Join-Path $Root 'build\windows\icon.ico'
$Template = Join-Path $Root 'build\windows\installer.nsi'
$Output = [IO.Path]::GetFullPath($Output)
$OutDir = Split-Path -Parent $Output
$MakensisCommand = Get-Command makensis -ErrorAction SilentlyContinue | Select-Object -First 1
$Makensis = if ($MakensisCommand) { $MakensisCommand.Source } else { '' }
if (-not $Makensis) {
    $Makensis = @(
        "${env:ProgramFiles}\NSIS\makensis.exe",
        "${env:ProgramFiles(x86)}\NSIS\makensis.exe"
    ) | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
}
if (-not $Makensis) { throw 'makensis is required to build the Windows installer' }
if (-not (Test-Path -LiteralPath $Source)) { throw "Built executable not found at $Source" }
if (-not (Test-Path -LiteralPath $Icon)) { throw "Installer icon not found at $Icon" }
if (-not (Test-Path -LiteralPath $Template)) { throw "Installer template not found at $Template" }

$Manifest = Join-Path $Root 'build\windows\wails.exe.uiaccess.manifest'
if (-not (Test-Path -LiteralPath $Manifest)) { throw "uiAccess manifest not found at $Manifest" }
$DevSign = Join-Path $Root 'scripts\dev-sign.ps1'

& $DevSign -Command uiaccess -Path $Source -Manifest $Manifest

New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
& $Makensis "/DVERSION=$Version" "/DSOURCE_EXE=$Source" "/DSOURCE_ICON=$Icon" "/DOUT_FILE=$Output" $Template
if ($LASTEXITCODE -ne 0) { throw "makensis failed with exit code $LASTEXITCODE" }
& $DevSign -Command sign -Path $Output
