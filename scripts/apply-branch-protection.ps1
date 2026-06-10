param(
    [Parameter(Mandatory = $true)]
    [string]$Owner,

    [Parameter(Mandatory = $true)]
    [string]$Repo,

    [string[]]$Branches = @("master", "main"),

    [string]$Token = $env:GITHUB_TOKEN,

    [string[]]$RequiredChecks = @("go-ci")
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Token)) {
    throw "Set GITHUB_TOKEN with admin rights for branch protection."
}

$headers = @{
    Authorization          = "Bearer $Token"
    Accept                 = "application/vnd.github+json"
    "X-GitHub-Api-Version" = "2022-11-28"
}

try {
    Invoke-RestMethod -Method Get -Uri "https://api.github.com/user" -Headers $headers | Out-Null
}
catch {
    throw "Invalid GITHUB_TOKEN or insufficient API access."
}

$appliedCount = 0

foreach ($branch in $Branches) {
    if ([string]::IsNullOrWhiteSpace($branch)) {
        continue
    }

    $payload = @{
        required_status_checks = @{
            strict   = $true
            contexts = $RequiredChecks
        }
        enforce_admins                  = $true
        required_pull_request_reviews   = @{
            dismiss_stale_reviews           = $true
            require_code_owner_reviews      = $false
            required_approving_review_count = 1
        }
        restrictions                    = $null
        required_conversation_resolution = $true
        allow_force_pushes              = $false
        allow_deletions                 = $false
        block_creations                 = $false
        required_linear_history         = $true
        lock_branch                     = $false
        allow_fork_syncing              = $true
    } | ConvertTo-Json -Depth 10

    $protectionUrl = "https://api.github.com/repos/$Owner/$Repo/branches/$branch/protection"

    try {
        Invoke-RestMethod -Method Put -Uri $protectionUrl -Headers $headers -Body $payload -ContentType "application/json" | Out-Null
        Write-Host "Applied protection to branch: $branch"
        $appliedCount++
    }
    catch {
        $statusCode = $null
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }

        if ($statusCode -eq 404) {
            Write-Host "Skipping missing branch: $branch"
            continue
        }

        throw "Failed to apply branch protection to '$branch' (HTTP $statusCode)."
    }
}

if ($appliedCount -eq 0) {
    throw "No branch protection rules were applied. Check branch names and token permissions."
}
