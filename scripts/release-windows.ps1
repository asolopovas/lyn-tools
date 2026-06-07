[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidateSet('bump', 'release', 'package-windows', 'package-deb', 'publish-assets')]
    [string]$Command,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Args
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Get-CurrentVersion {
    if (Test-Path -LiteralPath 'VERSION') {
        return (Get-Content -LiteralPath 'VERSION' -Raw).Trim()
    }
    $Wails = Get-Content -LiteralPath 'wails.json' -Raw | ConvertFrom-Json
    if ($Wails.info.productVersion) { return [string]$Wails.info.productVersion }
    return '0.1.0'
}

function Test-Version([string]$Version) {
    return $Version -match '^[0-9]+\.[0-9]+\.[0-9]+$'
}

function Get-NextVersion([string]$Current, [string]$Bump) {
    $Current = $Current.TrimStart('v')
    $Parts = $Current.Split('.')
    if ($Parts.Count -ne 3) { throw "release: invalid current version $Current" }
    $Major = [int]$Parts[0]
    $Minor = [int]$Parts[1]
    $Patch = [int]$Parts[2]
    switch -Regex ($Bump) {
        '^patch$' { return "$Major.$Minor.$($Patch + 1)" }
        '^minor$' { return "$Major.$($Minor + 1).0" }
        '^major$' { return "$($Major + 1).0.0" }
        '^v?[0-9]+\.[0-9]+\.[0-9]+$' { return $Bump.TrimStart('v') }
        default { throw "release: invalid bump $Bump" }
    }
}

function Sync-VersionFiles([string]$OldVersion, [string]$NewVersion) {
    Set-Content -LiteralPath 'VERSION' -Value $NewVersion -Encoding UTF8
    $Wails = Get-Content -LiteralPath 'wails.json' -Raw | ConvertFrom-Json
    if (-not $Wails.info) {
        $Wails | Add-Member -MemberType NoteProperty -Name info -Value ([pscustomobject]@{})
    }
    $Wails.info.productVersion = $NewVersion
    $Wails | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath 'wails.json' -Encoding UTF8
    if (Test-Path -LiteralPath 'README.md') {
        $Text = Get-Content -LiteralPath 'README.md' -Raw
        $Text.Replace($OldVersion, $NewVersion) | Set-Content -LiteralPath 'README.md' -Encoding UTF8
    }
}

function Assert-Clean {
    & git diff --quiet
    if ($LASTEXITCODE -ne 0) { throw 'release: tracked files changed; commit or stash first' }
    & git diff --cached --quiet
    if ($LASTEXITCODE -ne 0) { throw 'release: staged changes exist; commit or stash first' }
}

function Write-Checksums([string]$OutDir) {
    $Files = Get-ChildItem -LiteralPath $OutDir -File | Where-Object { $_.Extension -in @('.exe', '.deb') }
    $Lines = foreach ($File in $Files) {
        $Hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $File.FullName).Hash.ToLowerInvariant()
        "$Hash  $($File.Name)"
    }
    Set-Content -LiteralPath (Join-Path $OutDir 'SHA256SUMS') -Value $Lines -Encoding ASCII
}

function Get-ArchName {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        'ARM64' { return 'arm64' }
        default { return 'x64' }
    }
}

function Build-WindowsPackage([string]$Version) {
    $Tag = "v$Version"
    $Arch = Get-ArchName
    $OutDir = Join-Path 'releases' $Tag
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
    $Output = Join-Path $OutDir "lyn-$Tag-windows-$Arch-setup.exe"
    & (Join-Path $PWD 'scripts/package-windows.ps1') -Root $PWD -Version $Version -Output $Output | Out-Host
    if ($LASTEXITCODE -ne 0) { throw "package-windows.ps1 failed with exit code $LASTEXITCODE" }
    Write-Checksums $OutDir
    return $OutDir
}

function Build-DebPackage([string]$Version) {
    $Tag = "v$Version"
    $Arch = Get-ArchName
    $OutDir = Join-Path 'releases' $Tag
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
    $Root = (Resolve-Path -LiteralPath '.').Path
    $Image = if ($env:LYN_DEB_BUILD_IMAGE) { $env:LYN_DEB_BUILD_IMAGE } else { 'golang:1.26-bookworm' }
    $Output = "/work/releases/$Tag/lyn-$Tag-linux-$Arch.deb"
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'release: docker is required to build the Linux .deb on Windows' }
    $BuildScript = @"
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
export PATH="/usr/local/go/bin:`$PATH"
apt-get update
apt-get install -y --no-install-recommends build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev libayatana-appindicator3-dev dpkg-dev
rm -rf /var/lib/apt/lists/*
sed -i 's/\r$//' scripts/package-deb.sh
go build -mod=readonly -tags "desktop,production" -ldflags "-w -s" -o build/bin/lyn .
bash scripts/package-deb.sh /work "$Version" "$Arch" "$Output"
"@
    $BuildScript = $BuildScript -replace "`r`n", "`n"
    & docker run --rm --mount "type=bind,source=$Root,target=/work" -w /work $Image bash -lc $BuildScript
    if ($LASTEXITCODE -ne 0) { throw "docker deb build failed with exit code $LASTEXITCODE" }
    Write-Checksums $OutDir
    return $OutDir
}

function Publish-Assets([string[]]$CommandArgs) {
    $Version = Get-CurrentVersion
    $Tag = "v$($Version.TrimStart('v'))"
    if ($CommandArgs.Count -gt 0 -and $CommandArgs[0]) {
        $Tag = $CommandArgs[0]
    }
    $OutDir = Join-Path 'releases' $Tag
    if (-not (Test-Path -LiteralPath $OutDir)) { throw "release: missing asset directory $OutDir" }
    $NativeAssets = @(Get-ChildItem -LiteralPath $OutDir -File | Where-Object { $_.Extension -in @('.exe', '.deb') -or $_.Name -eq 'SHA256SUMS' })
    if ($NativeAssets.Count -eq 0) { throw "release: no native assets found in $OutDir" }
    $ExistingAssets = & gh release view $Tag --json assets --jq '.assets[].name'
    if ($LASTEXITCODE -ne 0) { throw "gh release view failed with exit code $LASTEXITCODE" }
    foreach ($Asset in $ExistingAssets) {
        if ($Asset.EndsWith('.tar.gz') -or $Asset -eq 'SHA256SUMS') {
            & gh release delete-asset $Tag $Asset --yes
            if ($LASTEXITCODE -ne 0) { throw "gh release delete-asset failed for $Asset with exit code $LASTEXITCODE" }
        }
    }
    $Paths = $NativeAssets | ForEach-Object { $_.FullName }
    & gh release upload $Tag @Paths --clobber
    if ($LASTEXITCODE -ne 0) { throw "gh release upload failed with exit code $LASTEXITCODE" }
    & gh release edit $Tag --latest
    if ($LASTEXITCODE -ne 0) { throw "gh release edit failed with exit code $LASTEXITCODE" }
}

function Invoke-Bump([string[]]$CommandArgs) {
    $Bump = if ($CommandArgs.Count -gt 0 -and $CommandArgs[0]) { $CommandArgs[0] } else { 'patch' }
    $Current = Get-CurrentVersion
    if (-not (Test-Version $Current.TrimStart('v'))) { throw "release: invalid current version $Current" }
    $Next = Get-NextVersion $Current $Bump
    Sync-VersionFiles $Current.TrimStart('v') $Next
    Write-Output $Next
}

function Invoke-Release([string[]]$CommandArgs) {
    $Bump = ''
    $Push = $true
    $Force = $false
    $DryRun = $false
    for ($I = 0; $I -lt $CommandArgs.Count; $I++) {
        switch -Regex ($CommandArgs[$I]) {
            '^--bump$' {
                if ($I + 1 -lt $CommandArgs.Count -and -not $CommandArgs[$I + 1].StartsWith('--')) {
                    $Bump = $CommandArgs[$I + 1]
                    $I++
                } else {
                    $Bump = 'patch'
                }
            }
            '^--bump=' { $Bump = $CommandArgs[$I].Substring(7) }
            '^--force$' { $Force = $true }
            '^--no-push$' { $Push = $false }
            '^--push$' { $Push = $true }
            '^--dry-run$' { $DryRun = $true }
            default { throw "release: unknown argument $($CommandArgs[$I])" }
        }
    }
    $Current = Get-CurrentVersion
    if (-not (Test-Version $Current.TrimStart('v'))) { throw "release: invalid current version $Current" }
    $Next = $Current.TrimStart('v')
    if ($Bump) { $Next = Get-NextVersion $Current $Bump }
    $Tag = "v$Next"
    if ($DryRun) {
        Write-Output "release: version=$Next tag=$Tag platform=windows assets=windows-installer,linux-deb-docker arch=$(Get-ArchName) push=$Push force=$Force"
        return
    }
    Assert-Clean
    if ($Bump) { Sync-VersionFiles $Current.TrimStart('v') $Next }
    & just check
    if ($LASTEXITCODE -ne 0) { throw "just check failed with exit code $LASTEXITCODE" }
    & just build
    if ($LASTEXITCODE -ne 0) { throw "just build failed with exit code $LASTEXITCODE" }
    $OutDir = Build-WindowsPackage $Next
    Build-DebPackage $Next | Out-Null
    Write-Checksums $OutDir
    if ($Bump) {
        & git add VERSION README.md wails.json frontend/dist
        if ($LASTEXITCODE -ne 0) { throw "git add failed with exit code $LASTEXITCODE" }
        & git commit -m "Release $Tag"
        if ($LASTEXITCODE -ne 0) { throw "git commit failed with exit code $LASTEXITCODE" }
    } else {
        & git diff --quiet
        if ($LASTEXITCODE -ne 0) { throw 'release: build changed tracked files; rerun with --bump or commit build output first' }
    }
    & git rev-parse -q --verify "refs/tags/$Tag" | Out-Null
    if ($LASTEXITCODE -eq 0) {
        if ($Force) {
            & git tag -d $Tag | Out-Null
        } else {
            throw "release: tag exists: $Tag; use --force to move it"
        }
    }
    & git tag $Tag
    if ($Push) {
        & git push origin HEAD
        if ($LASTEXITCODE -ne 0) { throw "git push failed with exit code $LASTEXITCODE" }
        if ($Force) {
            & git push origin "refs/tags/$Tag`:refs/tags/$Tag" --force
            & gh release delete $Tag --yes 2>$null
        } else {
            & git push origin $Tag
            if ($LASTEXITCODE -ne 0) { throw "git push tag failed with exit code $LASTEXITCODE" }
            & gh release view $Tag *> $null
            if ($LASTEXITCODE -eq 0) { throw "release: GitHub release exists: $Tag; use --force to replace" }
        }
        $ReleaseAssetPaths = Get-ChildItem -LiteralPath $OutDir -File | ForEach-Object { $_.FullName }
        & gh release create $Tag @ReleaseAssetPaths --title "Lyn $Tag" --notes "Lyn $Tag" --latest
        if ($LASTEXITCODE -ne 0) { throw "gh release create failed with exit code $LASTEXITCODE" }
    }
    Write-Output "release: $Tag ready in $OutDir"
}

switch ($Command) {
    'bump' { Invoke-Bump $Args }
    'package-windows' { Build-WindowsPackage (Get-CurrentVersion).TrimStart('v') | Write-Output }
    'package-deb' { Build-DebPackage (Get-CurrentVersion).TrimStart('v') | Write-Output }
    'publish-assets' { Publish-Assets $Args }
    'release' { Invoke-Release $Args }
}
