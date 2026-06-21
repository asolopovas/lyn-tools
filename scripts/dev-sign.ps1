[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('setup', 'sign', 'uiaccess')]
    [string]$Command,
    [string]$Path,
    [string]$Manifest,
    [string]$Subject = 'Lyn Code Signing'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Test-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    return ([Security.Principal.WindowsPrincipal]$identity).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Find-SdkTool {
    param([string]$Name)
    $command = Get-Command $Name -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($command) { return $command.Source }
    $roots = @("${env:ProgramFiles(x86)}\Windows Kits\10\bin", "${env:ProgramFiles}\Windows Kits\10\bin") | Where-Object { Test-Path -LiteralPath $_ }
    $found = Get-ChildItem -Path $roots -Recurse -Filter "$Name.exe" -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -match '\\x64\\' } |
        Sort-Object FullName -Descending |
        Select-Object -First 1
    if ($found) { return $found.FullName }
    throw "$Name.exe not found. Install the Windows SDK or add it to PATH."
}

function Invoke-Sign {
    param([string]$Target)
    if (-not (Test-Path -LiteralPath $Target)) { throw "File not found at $Target" }
    $cert = Get-ChildItem Cert:\LocalMachine\My | Where-Object { $_.Subject -eq "CN=$Subject" } | Select-Object -First 1
    if (-not $cert) { throw "Signing certificate CN=$Subject not found. Run 'just dev-sign-setup' first." }
    $signtool = Find-SdkTool signtool
    $signArgs = @('sign', '/sm', '/sha1', $cert.Thumbprint, '/fd', 'SHA256')
    if ($env:LYN_SIGN_TIMESTAMP_URL) { $signArgs += @('/tr', $env:LYN_SIGN_TIMESTAMP_URL, '/td', 'SHA256') }
    $signArgs += $Target
    & $signtool @signArgs
    if ($LASTEXITCODE -ne 0) { throw "signtool sign failed with exit code $LASTEXITCODE" }
    & $signtool verify /pa $Target
    if ($LASTEXITCODE -ne 0) { throw "signtool verify failed with exit code $LASTEXITCODE" }
    Write-Output "Signed $Target"
}

if ($Command -eq 'setup') {
    if (-not (Test-Administrator)) { throw 'dev-sign setup must run elevated; it writes machine certificate stores.' }
    $cert = Get-ChildItem Cert:\LocalMachine\My | Where-Object { $_.Subject -eq "CN=$Subject" } | Select-Object -First 1
    if (-not $cert) {
        $cert = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=$Subject" -CertStoreLocation Cert:\LocalMachine\My -KeyUsage DigitalSignature -KeyExportPolicy Exportable -KeyLength 2048 -NotAfter (Get-Date).AddYears(5)
    }
    $cerPath = Join-Path $env:TEMP 'lyn-code-signing.cer'
    Export-Certificate -Cert $cert -FilePath $cerPath | Out-Null
    Import-Certificate -FilePath $cerPath -CertStoreLocation Cert:\LocalMachine\Root | Out-Null
    Import-Certificate -FilePath $cerPath -CertStoreLocation Cert:\LocalMachine\TrustedPublisher | Out-Null
    Write-Output "Installed code-signing certificate CN=$Subject (thumbprint $($cert.Thumbprint))."
    return
}

if ($Command -eq 'uiaccess') {
    if (-not $Path) { throw 'uiaccess requires -Path to the executable.' }
    if (-not $Manifest) { throw 'uiaccess requires -Manifest to the uiAccess manifest.' }
    if (-not (Test-Path -LiteralPath $Path)) { throw "Executable not found at $Path" }
    if (-not (Test-Path -LiteralPath $Manifest)) { throw "Manifest not found at $Manifest" }
    $mt = Find-SdkTool mt
    & $mt -nologo -manifest $Manifest "-outputresource:$Path;#1"
    if ($LASTEXITCODE -ne 0) { throw "mt.exe failed to stamp the uiAccess manifest (exit $LASTEXITCODE)" }
    Invoke-Sign $Path
    return
}

if (-not $Path) { throw 'sign requires -Path to a file.' }
Invoke-Sign $Path
