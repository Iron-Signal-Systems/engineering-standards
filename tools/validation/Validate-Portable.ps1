[CmdletBinding()]
param()
$ErrorActionPreference = "Stop"
$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if (-not $repoRoot) {
    Write-Error "FAIL: not in a Git work tree"
    Write-Error "failure_code=ISRAS-PORTABLE-ENTRYPOINT-001"
    exit 1
}
$pythonPath = $env:ISRAS_PYTHON
if (-not $pythonPath) {
    $python = Get-Command python3 -ErrorAction SilentlyContinue
    if (-not $python) { $python = Get-Command python -ErrorAction SilentlyContinue }
    if (-not $python) {
        Write-Error "FAIL: Python 3 is required by the portable validator"
        Write-Error "failure_code=ISRAS-PORTABLE-ENTRYPOINT-002"
        exit 1
    }
    $pythonPath = $python.Source
}
& $pythonPath -I "$repoRoot/tools/isras/run_portable_validation.py" --repo-root "$repoRoot"
exit $LASTEXITCODE
