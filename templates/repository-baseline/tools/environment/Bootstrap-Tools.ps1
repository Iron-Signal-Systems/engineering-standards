[CmdletBinding()]
param(
    [string]$VenvPath
)
$ErrorActionPreference = "Stop"
$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if (-not $repoRoot) { throw "Not in a Git work tree." }
if (-not $VenvPath) { $VenvPath = Join-Path $repoRoot ".isras-tools-venv" }
$python = Get-Command python3 -ErrorAction SilentlyContinue
if (-not $python) { $python = Get-Command python -ErrorAction SilentlyContinue }
if (-not $python) { throw "Python 3 is required." }
& $python.Source -m venv $VenvPath
$venvPython = Join-Path $VenvPath "Scripts/python.exe"
& $venvPython -m pip install --upgrade pip
& $venvPython -m pip install -r (Join-Path $repoRoot "tools/requirements.txt")
Write-Host "ISRAS tool environment created at $VenvPath"
Write-Host "Set ISRAS_PYTHON=$venvPython to use it."
