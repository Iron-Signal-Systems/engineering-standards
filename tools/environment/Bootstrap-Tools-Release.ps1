[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Wheelhouse,
    [string]$VenvPath
)
$ErrorActionPreference = "Stop"
$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if (-not $repoRoot) { throw "Not in a Git work tree." }
if (-not $VenvPath) { $VenvPath = Join-Path $repoRoot ".isras-tools-venv" }
if (Test-Path -LiteralPath $VenvPath) {
    throw "Release tool environment path already exists: $VenvPath"
}
$Wheelhouse = (Resolve-Path -LiteralPath $Wheelhouse).Path

$python = Get-Command python3 -ErrorAction SilentlyContinue
if (-not $python) { $python = Get-Command python -ErrorAction SilentlyContinue }
if (-not $python) { throw "Python 3 is required." }

& $python.Source -I (Join-Path $repoRoot "tools\environment\verify_wheelhouse.py") `
    --repo-root $repoRoot `
    --wheelhouse $Wheelhouse
if ($LASTEXITCODE -ne 0) { throw "Wheelhouse verification failed." }

& $python.Source -I -m venv $VenvPath
if ($LASTEXITCODE -ne 0) { throw "Virtual environment creation failed." }
$venvPython = Join-Path $VenvPath "Scripts\python.exe"
$wheels = Join-Path $Wheelhouse "wheels"

$priorConfig = $env:PIP_CONFIG_FILE
$priorNoUserSite = $env:PYTHONNOUSERSITE
try {
    $env:PIP_CONFIG_FILE = "NUL"
    $env:PYTHONNOUSERSITE = "1"

    & $venvPython -I -m pip --isolated install `
        --disable-pip-version-check `
        --no-index `
        --no-cache-dir `
        --only-binary=:all: `
        --find-links $wheels `
        --require-hashes `
        --force-reinstall `
        --no-deps `
        -r (Join-Path $Wheelhouse "bootstrap-pip.lock")
    if ($LASTEXITCODE -ne 0) { throw "Pinned offline pip installation failed." }

    & $venvPython -I (Join-Path $repoRoot "tools\environment\clean_tool_venv.py") --keep pip
    if ($LASTEXITCODE -ne 0) { throw "Bootstrap-only distribution cleanup failed." }

    & $venvPython -I -m pip --isolated install `
        --disable-pip-version-check `
        --no-index `
        --no-cache-dir `
        --only-binary=:all: `
        --find-links $wheels `
        --require-hashes `
        -r (Join-Path $Wheelhouse "requirements.lock")
    if ($LASTEXITCODE -ne 0) { throw "Pinned offline dependency installation failed." }

    & $venvPython -I (Join-Path $repoRoot "tools\environment\record_tool_environment.py") `
        --bootstrap-mode release `
        --requirements (Join-Path $Wheelhouse "requirements.lock") `
        --bootstrap-lock (Join-Path $Wheelhouse "bootstrap-lock.json") `
        --wheelhouse-manifest (Join-Path $Wheelhouse "SHA512SUMS") `
        --output (Join-Path $VenvPath "isras-tool-environment.json")
    if ($LASTEXITCODE -ne 0) { throw "Tool environment recording failed." }
}
finally {
    $env:PIP_CONFIG_FILE = $priorConfig
    $env:PYTHONNOUSERSITE = $priorNoUserSite
}

Write-Host "ISRAS release tool environment created at $VenvPath"
Write-Host "Set ISRAS_PYTHON=$venvPython to use it."
