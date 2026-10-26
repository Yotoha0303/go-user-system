param(
    [string]$BaseUrl = "http://127.0.0.1:18082",
    [Parameter(Mandatory = $true)]
    [string]$AdminUsername,
    [Parameter(Mandatory = $true)]
    [string]$AdminPassword,
    [string]$OutputPath = "artifacts/test-runs/latest/acceptance-results.json"
)

$ErrorActionPreference = "Stop"
$results = [System.Collections.Generic.List[object]]::new()
$runSuffix = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds().ToString()
$refreshUri = [Uri]"$BaseUrl/api/v1/auth/refresh"

function Assert-Condition {
    param(
        [bool]$Condition,
        [string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Path,
        [object]$Body,
        [string]$AccessToken,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession
    )

    $parameters = @{
        Uri                  = "$BaseUrl$Path"
        Method               = $Method
        SkipHttpErrorCheck   = $true
        UseBasicParsing      = $true
    }

    if ($null -ne $Body) {
        $parameters.ContentType = "application/json"
        $parameters.Body = $Body | ConvertTo-Json -Compress
    }
    if (-not [string]::IsNullOrWhiteSpace($AccessToken)) {
        $parameters.Headers = @{ Authorization = "Bearer $AccessToken" }
    }
    if ($null -ne $WebSession) {
        $parameters.WebSession = $WebSession
    }

    $response = Invoke-WebRequest @parameters
    $payload = $null
    if (-not [string]::IsNullOrWhiteSpace($response.Content)) {
        try {
            $payload = $response.Content | ConvertFrom-Json
        } catch {
            $payload = $response.Content
        }
    }

    [pscustomobject]@{
        Status  = [int]$response.StatusCode
        Headers = $response.Headers
        Body    = $payload
    }
}

function Register-User {
    param(
        [string]$Username,
        [string]$Password
    )

    Invoke-Api -Method POST -Path "/api/v1/auth/register" -Body @{
        username = $Username
        password = $Password
    }
}

function Login-User {
    param(
        [string]$Username,
        [string]$Password,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession
    )

    Invoke-Api -Method POST -Path "/api/v1/auth/login" -Body @{
        username = $Username
        password = $Password
    } -WebSession $WebSession
}

function Copy-RefreshSession {
    param([Microsoft.PowerShell.Commands.WebRequestSession]$Source)

    $cookies = $Source.Cookies.GetCookies($refreshUri)
    $refreshCookie = $cookies | Where-Object Name -eq "refresh_token" | Select-Object -First 1
    Assert-Condition ($null -ne $refreshCookie) "refresh_token cookie is missing"

    $copy = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $cookie = [System.Net.Cookie]::new(
        $refreshCookie.Name,
        $refreshCookie.Value,
        $refreshCookie.Path
    )
    $copy.Cookies.Add($refreshUri, $cookie)
    $copy
}

function Invoke-Case {
    param(
        [string]$Id,
        [scriptblock]$Test
    )

    $started = Get-Date
    try {
        $detail = & $Test
        $results.Add([pscustomobject]@{
            id          = $Id
            result      = "PASS"
            detail      = [string]$detail
            duration_ms = [int]((Get-Date) - $started).TotalMilliseconds
        })
        Write-Host "PASS $Id"
    } catch {
        $results.Add([pscustomobject]@{
            id          = $Id
            result      = "FAIL"
            detail      = $_.Exception.Message
            duration_ms = [int]((Get-Date) - $started).TotalMilliseconds
        })
        Write-Host "FAIL $Id - $($_.Exception.Message)"
    }
}

$userAPassword = "Aa1!accept-$runSuffix"
$userANewPassword = "Bb2!changed-$runSuffix"
$wrongPassword = "Wrong!$runSuffix"
$userA = "accept_a_$runSuffix"
$userB = "accept_b_$runSuffix"
$replayUser = "accept_replay_$runSuffix"
$logoutUser = "accept_logout_$runSuffix"
$rateUser = "accept_rate_$runSuffix"

Invoke-Case "HEALTH-01" {
    foreach ($path in @("/ping", "/livez", "/readyz", "/version", "/metrics")) {
        $response = Invoke-Api -Method GET -Path $path
        Assert-Condition ($response.Status -eq 200) "$path returned $($response.Status)"
    }
    "five system endpoints returned 200"
}

Invoke-Case "REG-01" {
    $response = Register-User -Username $userA -Password $userAPassword
    Assert-Condition ($response.Status -eq 200) "valid registration returned $($response.Status)"
    $responseB = Register-User -Username $userB -Password $userAPassword
    Assert-Condition ($responseB.Status -eq 200) "second user registration returned $($responseB.Status)"
    "two regular users registered"
}

Invoke-Case "REG-03" {
    $short = Register-User -Username "short_$runSuffix" -Password "12345678901"
    Assert-Condition ($short.Status -eq 400) "11-character password returned $($short.Status)"
    $long = Register-User -Username "long_$runSuffix" -Password ("x" * 73)
    Assert-Condition ($long.Status -eq 400) "73-byte password returned $($long.Status)"
    $duplicate = Register-User -Username $userA -Password $userAPassword
    Assert-Condition ($duplicate.Status -in @(400, 409)) "duplicate username returned $($duplicate.Status)"
    "password boundaries and duplicate username rejected"
}

Invoke-Case "AUTH-01" {
    $wrong = Login-User -Username $userA -Password $wrongPassword -WebSession ([Microsoft.PowerShell.Commands.WebRequestSession]::new())
    $missing = Login-User -Username "missing_$runSuffix" -Password $wrongPassword -WebSession ([Microsoft.PowerShell.Commands.WebRequestSession]::new())
    Assert-Condition ($wrong.Status -eq $missing.Status) "wrong and missing users returned different status"
    Assert-Condition ($wrong.Body.code -eq $missing.Body.code) "wrong and missing users returned different code"
    Assert-Condition ($wrong.Body.msg -eq $missing.Body.msg) "wrong and missing users returned different message"
    "invalid credentials do not disclose account existence"
}

Invoke-Case "PROFILE-01" {
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $login = Login-User -Username $userA -Password $userAPassword -WebSession $session
    Assert-Condition ($login.Status -eq 200) "login returned $($login.Status)"
    $token = [string]$login.Body.data.access_token
    $update = Invoke-Api -Method PUT -Path "/api/v1/users/me/profile" -Body @{ nickname = "Acceptance User A" } -AccessToken $token
    Assert-Condition ($update.Status -eq 200) "profile update returned $($update.Status)"
    $profile = Invoke-Api -Method GET -Path "/api/v1/users/me" -AccessToken $token
    Assert-Condition ($profile.Status -eq 200) "profile read returned $($profile.Status)"
    Assert-Condition ($profile.Body.data.nickname -eq "Acceptance User A") "nickname was not persisted"
    "profile persisted for current user"
}

Invoke-Case "RBAC-01" {
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $login = Login-User -Username $userA -Password $userAPassword -WebSession $session
    $token = [string]$login.Body.data.access_token
    $authorization = Invoke-Api -Method GET -Path "/api/v1/users/me/authorization" -AccessToken $token
    Assert-Condition ($authorization.Status -eq 200) "authorization returned $($authorization.Status)"
    Assert-Condition (@($authorization.Body.data.role_codes) -contains "user") "regular user role is missing"
    Assert-Condition (-not (@($authorization.Body.data.role_codes) -contains "admin")) "regular user unexpectedly has admin role"
    $adminAttempt = Invoke-Api -Method GET -Path "/api/v1/admin/roles" -AccessToken $token
    Assert-Condition ($adminAttempt.Status -eq 403) "regular user admin request returned $($adminAttempt.Status)"
    "regular user has user role and receives 403 from admin API"
}

Invoke-Case "SES-03" {
    $register = Register-User -Username $replayUser -Password $userAPassword
    Assert-Condition ($register.Status -eq 200) "replay user registration returned $($register.Status)"
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $login = Login-User -Username $replayUser -Password $userAPassword -WebSession $session
    Assert-Condition ($login.Status -eq 200) "replay user login returned $($login.Status)"
    $oldCookieSession = Copy-RefreshSession -Source $session
    $rotate = Invoke-Api -Method POST -Path "/api/v1/auth/refresh" -WebSession $session
    Assert-Condition ($rotate.Status -eq 200) "first refresh returned $($rotate.Status)"
    $replay = Invoke-Api -Method POST -Path "/api/v1/auth/refresh" -WebSession $oldCookieSession
    Assert-Condition ($replay.Status -eq 401) "old refresh replay returned $($replay.Status)"
    $family = Invoke-Api -Method POST -Path "/api/v1/auth/refresh" -WebSession $session
    Assert-Condition ($family.Status -eq 401) "token family remained active after replay"
    "old refresh rejected and token family revoked"
}

Invoke-Case "PWD-01" {
    $sessionA = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $sessionB = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $loginA = Login-User -Username $userA -Password $userAPassword -WebSession $sessionA
    $loginB = Login-User -Username $userA -Password $userAPassword -WebSession $sessionB
    Assert-Condition ($loginA.Status -eq 200 -and $loginB.Status -eq 200) "two-session login failed"
    $tokenA = [string]$loginA.Body.data.access_token
    $tokenB = [string]$loginB.Body.data.access_token
    $change = Invoke-Api -Method PATCH -Path "/api/v1/users/me/update/password" -Body @{
        old_password = $userAPassword
        new_password = $userANewPassword
    } -AccessToken $tokenA
    Assert-Condition ($change.Status -eq 200) "password change returned $($change.Status)"
    $oldAccessA = Invoke-Api -Method GET -Path "/api/v1/users/me" -AccessToken $tokenA
    $oldAccessB = Invoke-Api -Method GET -Path "/api/v1/users/me" -AccessToken $tokenB
    Assert-Condition ($oldAccessA.Status -eq 401 -and $oldAccessB.Status -eq 401) "an old access token remained valid"
    $oldRefreshA = Invoke-Api -Method POST -Path "/api/v1/auth/refresh" -WebSession $sessionA
    $oldRefreshB = Invoke-Api -Method POST -Path "/api/v1/auth/refresh" -WebSession $sessionB
    Assert-Condition ($oldRefreshA.Status -eq 401 -and $oldRefreshB.Status -eq 401) "an old refresh token remained valid"
    $oldLogin = Login-User -Username $userA -Password $userAPassword -WebSession ([Microsoft.PowerShell.Commands.WebRequestSession]::new())
    $newLogin = Login-User -Username $userA -Password $userANewPassword -WebSession ([Microsoft.PowerShell.Commands.WebRequestSession]::new())
    Assert-Condition ($oldLogin.Status -eq 401) "old password returned $($oldLogin.Status)"
    Assert-Condition ($newLogin.Status -eq 200) "new password returned $($newLogin.Status)"
    "old access, refresh and password invalidated across two sessions"
}

Invoke-Case "SES-04" {
    $register = Register-User -Username $logoutUser -Password $userAPassword
    Assert-Condition ($register.Status -eq 200) "logout user registration returned $($register.Status)"
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $login = Login-User -Username $logoutUser -Password $userAPassword -WebSession $session
    $token = [string]$login.Body.data.access_token
    $logout = Invoke-Api -Method POST -Path "/api/v1/auth/logout" -AccessToken $token -WebSession $session
    Assert-Condition ($logout.Status -eq 200) "logout returned $($logout.Status)"
    $oldAccess = Invoke-Api -Method GET -Path "/api/v1/users/me" -AccessToken $token
    $oldRefresh = Invoke-Api -Method POST -Path "/api/v1/auth/refresh" -WebSession $session
    Assert-Condition ($oldAccess.Status -eq 401) "logged-out access token returned $($oldAccess.Status)"
    Assert-Condition ($oldRefresh.Status -eq 401) "logged-out refresh returned $($oldRefresh.Status)"
    "logout revoked current access and refresh"
}

Invoke-Case "RBAC-02" {
    $adminSession = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $adminLogin = Login-User -Username $AdminUsername -Password $AdminPassword -WebSession $adminSession
    Assert-Condition ($adminLogin.Status -eq 200) "admin login returned $($adminLogin.Status)"
    $adminToken = [string]$adminLogin.Body.data.access_token
    $roles = Invoke-Api -Method GET -Path "/api/v1/admin/roles" -AccessToken $adminToken
    $permissions = Invoke-Api -Method GET -Path "/api/v1/admin/permissions" -AccessToken $adminToken
    Assert-Condition ($roles.Status -eq 200) "role list returned $($roles.Status)"
    Assert-Condition ($permissions.Status -eq 200) "permission list returned $($permissions.Status)"

    $userBSession = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $userBLogin = Login-User -Username $userB -Password $userAPassword -WebSession $userBSession
    $userBToken = [string]$userBLogin.Body.data.access_token
    $userBProfile = Invoke-Api -Method GET -Path "/api/v1/users/me" -AccessToken $userBToken
    $userBId = [int64]$userBProfile.Body.data.id
    Assert-Condition ($userBId -gt 0) "user B id is missing"
    $assign = Invoke-Api -Method PUT -Path "/api/v1/admin/users/$userBId/roles" -Body @{
        role_codes = @("user", "admin")
    } -AccessToken $adminToken
    Assert-Condition ($assign.Status -eq 200) "role assignment returned $($assign.Status)"
    $updatedAuthorization = Invoke-Api -Method GET -Path "/api/v1/users/me/authorization" -AccessToken $userBToken
    Assert-Condition (@($updatedAuthorization.Body.data.role_codes) -contains "admin") "assigned admin role is missing"
    "admin listed RBAC data and assigned user B roles"
}

# Run the shared-IP rate-limit case last because it intentionally mutates
# Redis state that can reject otherwise valid logins within the configured window.
Invoke-Case "AUTH-02" {
    $statuses = @()
    $lastResponse = $null
    for ($attempt = 1; $attempt -le 5; $attempt++) {
        $lastResponse = Login-User -Username $rateUser -Password $wrongPassword -WebSession ([Microsoft.PowerShell.Commands.WebRequestSession]::new())
        $statuses += $lastResponse.Status
    }
    Assert-Condition ($statuses[-1] -eq 429) "threshold attempt returned $($statuses[-1]) instead of 429"
    Assert-Condition (-not [string]::IsNullOrWhiteSpace([string]$lastResponse.Headers["Retry-After"])) "Retry-After is missing"
    "account limit statuses: $($statuses -join ',')"
}

$outputDirectory = Split-Path -Parent $OutputPath
if (-not [string]::IsNullOrWhiteSpace($outputDirectory)) {
    New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
}

$summary = [pscustomobject]@{
    generated_at = (Get-Date).ToString("o")
    base_url      = $BaseUrl
    total         = $results.Count
    passed        = @($results | Where-Object result -eq "PASS").Count
    failed        = @($results | Where-Object result -eq "FAIL").Count
    results       = $results
}
$summary | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $OutputPath -Encoding utf8

Write-Host "Acceptance summary: $($summary.passed)/$($summary.total) passed"
if ($summary.failed -gt 0) {
    exit 1
}
