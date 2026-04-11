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

foreach ($branch in $Branches) {
    $branchUrl = "https://api.github.com/repos/$Owner/$Repo/branches/$branch"

    try {
        Invoke-RestMethod -Method Get -Uri $branchUrl -Headers $headers | Out-Null
    }
    catch {
        Write-Host "Skipping missing branch: $branch"
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
    Invoke-RestMethod -Method Put -Uri $protectionUrl -Headers $headers -Body $payload -ContentType "application/json" | Out-Null
    Write-Host "Applied protection to branch: $branch"
}
