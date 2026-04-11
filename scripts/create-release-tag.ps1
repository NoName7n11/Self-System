param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

if ($Version -notmatch '^v\d+\.\d+\.\d+$') {
    throw "Version must match vMAJOR.MINOR.PATCH (example: v0.1.0)."
}

$dirty = & git status --porcelain
if ($LASTEXITCODE -ne 0) {
    throw "Failed to read git status."
}
if ($dirty) {
    throw "Working tree is dirty. Commit or stash changes before tagging."
}

& git rev-parse -q --verify "refs/tags/$Version" *> $null
if ($LASTEXITCODE -eq 0) {
    throw "Tag $Version already exists."
}

if (-not $SkipTests) {
    & go test ./...
    if ($LASTEXITCODE -ne 0) {
        throw "Tests failed. Tag creation aborted."
    }
}

& git tag -a $Version -m "Release $Version"
if ($LASTEXITCODE -ne 0) {
    throw "Failed to create tag $Version."
}

& git push origin $Version
if ($LASTEXITCODE -ne 0) {
    throw "Tag created locally but push failed."
}

Write-Host "Release tag $Version created and pushed."
