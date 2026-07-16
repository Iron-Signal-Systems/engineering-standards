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

$requirements = Join-Path $repoRoot "tools\requirements.txt"
& $python.Source -m venv $VenvPath
$venvPython = Join-Path $VenvPath "Scripts\python.exe"

# Developer bootstrap does not implicitly upgrade pip. Release evidence must
# use Bootstrap-Tools-Release.ps1 with a reviewed wheelhouse.
& $venvPython -m pip install --disable-pip-version-check -r $requirements
if ($LASTEXITCODE -ne 0) { throw "Tool dependency installation failed." }

& $venvPython (Join-Path $repoRoot "tools\environment\record_tool_environment.py") `
    --bootstrap-mode developer `
    --requirements $requirements `
    --output (Join-Path $VenvPath "isras-tool-environment.json")
if ($LASTEXITCODE -ne 0) { throw "Tool environment recording failed." }

Write-Host "ISRAS developer tool environment created at $VenvPath"
Write-Host "Set ISRAS_PYTHON=$venvPython to use it."
Write-Host "NOTE: developer bootstrap is not release-assurance evidence."
