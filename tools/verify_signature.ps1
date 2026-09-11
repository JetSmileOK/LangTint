param([Parameter(Mandatory=$true)][string]$Stage,
      [Parameter(Mandatory=$true)][string]$SignedExe)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$root = Split-Path -Parent $PSScriptRoot
$config = Get-Content -LiteralPath (Join-Path $root 'packaging/release.json') -Raw | ConvertFrom-Json
$expected = $env:SIGNPATH_CERT_THUMBPRINT
if ($expected -notmatch '^[0-9A-Fa-f]{40}$') { throw 'Expected certificate thumbprint is not configured' }
$sig = Get-AuthenticodeSignature -LiteralPath $SignedExe
if ($sig.Status -ne 'Valid') { throw "Signature invalid: $($sig.Status)" }
if ($null -eq $sig.SignerCertificate -or $sig.SignerCertificate.Thumbprint -ne $expected) { throw 'Unexpected signing certificate' }
if ($null -eq $sig.TimeStamperCertificate) { throw 'Timestamp certificate missing' }
$info = [Diagnostics.FileVersionInfo]::GetVersionInfo((Resolve-Path -LiteralPath $SignedExe).Path)
if ($info.ProductName -ne 'LangTint' -or $info.ProductVersion -ne $config.version -or $info.FileVersion -ne $config.version) { throw 'Signed product/version metadata mismatch' }
if ($info.OriginalFilename -ne $config.exe) { throw 'OriginalFilename mismatch' }
$dest = Join-Path $Stage $config.exe
Copy-Item -LiteralPath $SignedExe -Destination $dest -Force
$hash = (Get-FileHash -LiteralPath $dest -Algorithm SHA256).Hash.ToLowerInvariant()
@{status='Valid';sha256=$hash;thumbprint=$sig.SignerCertificate.Thumbprint;subject=$sig.SignerCertificate.Subject} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Stage 'SIGNATURE.json') -Encoding utf8
$metadataPath = Join-Path $Stage 'RELEASE.json'
$metadata = Get-Content -LiteralPath $metadataPath -Raw | ConvertFrom-Json
$metadata.signing = 'signed'
$metadata | ConvertTo-Json | Set-Content -LiteralPath $metadataPath -Encoding utf8
