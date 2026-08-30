param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("Redis", "MySQL")]
    [string]$Scenario,
    [string]$BaseUrl = "http://127.0.0.1:18082",
    [string]$PrometheusUrl = "http://127.0.0.1:19090",
    [Parameter(Mandatory = $true)]
    [string]$AdminUsername,
    [Parameter(Mandatory = $true)]
    [string]$AdminPassword,
    [string]$OutputPath = "artifacts/test-runs/latest/fault-result.json",
    [switch]$WaitForAlert
)

$ErrorActionPreference = "Stop"
$compose = @(
    "compose",
    "-f", "compose.yaml",
    "-f", "compose.test.yaml",
    "-f", "compose.observability.yaml",
    "-p", "go-user-system-test"
)
$service = $Scenario.ToLowerInvariant()
$startedAt = Get-Date
$readyDuring = 0
$liveDuring = 0
$protectedDuring = 0
$readyAfter = 0
$alertFired = $false
$alertCleared = $false

function Get-HttpStatus {
    param(
        [string]$Uri,
        [hashtable]$Headers
    )

    $parameters = @{
        Uri                = $Uri
        UseBasicParsing    = $true
        SkipHttpErrorCheck = $true
    }
    if ($null -ne $Headers) {
        $parameters.Headers = $Headers
    }
    [int](Invoke-WebRequest @parameters).StatusCode
}

function Get-MatchingAlert {
    $alerts = (Invoke-RestMethod "$PrometheusUrl/api/v1/alerts").data.alerts
    @($alerts | Where-Object {
        $_.labels.alertname -eq "GoUserSystemNotReady" -and $_.state -eq "firing"
    }).Count -gt 0
}

$loginBody = @{
    username = $AdminUsername
    password = $AdminPassword
} | ConvertTo-Json -Compress
$login = Invoke-RestMethod -Method POST -Uri "$BaseUrl/api/v1/auth/login" -ContentType "application/json" -Body $loginBody
$headers = @{ Authorization = "Bearer $([string]$login.data.access_token)" }

try {
    docker @compose stop $service | Out-Null
    Start-Sleep -Seconds 3

    $readyDuring = Get-HttpStatus "$BaseUrl/readyz"
    $liveDuring = Get-HttpStatus "$BaseUrl/livez"
    $protectedDuring = Get-HttpStatus "$BaseUrl/api/v1/users/me" $headers
    Write-Host "$Scenario down: ready=$readyDuring live=$liveDuring protected=$protectedDuring"

    if ($readyDuring -eq 200) {
        throw "$Scenario outage did not make readiness fail"
    }
    if ($liveDuring -ne 200) {
        throw "$Scenario outage made liveness fail"
    }
    if ($protectedDuring -eq 200) {
        throw "$Scenario outage allowed a protected request"
    }

    if ($WaitForAlert) {
        for ($attempt = 1; $attempt -le 14; $attempt++) {
            Start-Sleep -Seconds 15
            if (Get-MatchingAlert) {
                $alertFired = $true
                Write-Host "GoUserSystemNotReady firing after $($attempt * 15) seconds"
                break
            }
            if (($attempt % 2) -eq 0) {
                Write-Host "GoUserSystemNotReady pending or absent after $($attempt * 15) seconds"
            }
        }
        if (-not $alertFired) {
            throw "GoUserSystemNotReady did not fire within 210 seconds"
        }
    }
} finally {
    docker @compose start $service | Out-Null
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        Start-Sleep -Seconds 2
        $readyAfter = Get-HttpStatus "$BaseUrl/readyz"
        if ($readyAfter -eq 200) {
            break
        }
    }
    Write-Host "$Scenario recovery: ready=$readyAfter"
}

if ($readyAfter -ne 200) {
    throw "$Scenario recovery readiness timeout"
}

if ($WaitForAlert) {
    for ($attempt = 1; $attempt -le 8; $attempt++) {
        Start-Sleep -Seconds 15
        if (-not (Get-MatchingAlert)) {
            $alertCleared = $true
            Write-Host "GoUserSystemNotReady cleared after $($attempt * 15) recovery seconds"
            break
        }
    }
    if (-not $alertCleared) {
        throw "GoUserSystemNotReady did not clear after recovery"
    }
}

$outputDirectory = Split-Path -Parent $OutputPath
if (-not [string]::IsNullOrWhiteSpace($outputDirectory)) {
    New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
}

[pscustomobject]@{
    generated_at        = (Get-Date).ToString("o")
    scenario            = $Scenario
    ready_during        = $readyDuring
    live_during         = $liveDuring
    protected_during    = $protectedDuring
    ready_after         = $readyAfter
    alert_wait_enabled  = [bool]$WaitForAlert
    alert_fired         = $alertFired
    alert_cleared       = $alertCleared
    duration_seconds    = [int]((Get-Date) - $startedAt).TotalSeconds
} | ConvertTo-Json | Set-Content -LiteralPath $OutputPath -Encoding utf8

