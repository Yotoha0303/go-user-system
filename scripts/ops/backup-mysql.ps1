[CmdletBinding()]
param(
    [string]$ProjectDirectory = "",
    [string]$OutputDirectory = "",
    [string]$ComposeProjectName = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-Compose {
    param(
        [Parameter(Mandatory)]
        [string[]]$Arguments
    )

    $composeArguments = @("compose", "--project-directory", $ProjectDirectory)
    if (-not [string]::IsNullOrWhiteSpace($ComposeProjectName)) {
        $composeArguments += @("--project-name", $ComposeProjectName)
    }

    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $output = & docker @composeArguments @Arguments 2>&1
        $exitCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    if ($exitCode -ne 0) {
        throw "docker compose $($Arguments -join ' ') failed: $($output -join [Environment]::NewLine)"
    }
    return $output
}

Get-Command docker -ErrorAction Stop | Out-Null
if ([string]::IsNullOrWhiteSpace($ProjectDirectory)) {
    $ProjectDirectory = Join-Path $PSScriptRoot "../.."
}
$ProjectDirectory = (Resolve-Path $ProjectDirectory).Path
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $ProjectDirectory "backups"
}
$OutputDirectory = [System.IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Force $OutputDirectory | Out-Null

$runningServices = @(Invoke-Compose -Arguments @("ps", "--status", "running", "--services"))
if ($runningServices -notcontains "mysql") {
    throw "the Compose mysql service is not running"
}

$createdAt = (Get-Date).ToUniversalTime()
$stamp = $createdAt.ToString("yyyyMMdd-HHmmss")
$containerPath = "/tmp/go_user_system-$stamp.sql"
$backupPath = Join-Path $OutputDirectory "go_user_system-$stamp.sql"
$manifestPath = "$backupPath.manifest.json"
$dumpCommand = 'umask 077; MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqldump -uroot --single-transaction --routines --triggers --events --set-gtid-purged=OFF go_user_system > ' + $containerPath

try {
    Invoke-Compose -Arguments @("exec", "-T", "mysql", "sh", "-c", $dumpCommand) | Out-Null
    $containerHashOutput = @(Invoke-Compose -Arguments @("exec", "-T", "mysql", "sha256sum", $containerPath))
    $containerHash = (($containerHashOutput -join " ") -split "\s+")[0].ToLowerInvariant()

    Invoke-Compose -Arguments @("cp", "mysql:$containerPath", $backupPath) | Out-Null
    $file = Get-Item -LiteralPath $backupPath
    if ($file.Length -le 0) {
        throw "backup file is empty: $backupPath"
    }
    $hostHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $backupPath).Hash.ToLowerInvariant()
    if ($hostHash -ne $containerHash) {
        throw "backup checksum mismatch between container and host"
    }

    $commit = "unknown"
    $gitCommand = Get-Command git -ErrorAction SilentlyContinue
    if ($null -ne $gitCommand) {
        $previousErrorActionPreference = $ErrorActionPreference
        $ErrorActionPreference = "Continue"
        try {
            $commitOutput = & $gitCommand.Source -C $ProjectDirectory rev-parse HEAD 2>$null
            $gitExitCode = $LASTEXITCODE
        }
        finally {
            $ErrorActionPreference = $previousErrorActionPreference
        }
        if ($gitExitCode -eq 0) {
            $commit = ($commitOutput | Select-Object -First 1).Trim()
        }
    }
    $manifest = [ordered]@{
        schema_version   = 1
        source           = "docker-compose"
        compose_project  = $ComposeProjectName
        database         = "go_user_system"
        created_at_utc   = $createdAt.ToString("o")
        application_commit = $commit
        file             = $file.Name
        size_bytes       = $file.Length
        sha256           = $hostHash
    }
    $manifest | ConvertTo-Json | Set-Content -LiteralPath $manifestPath -Encoding utf8

    [pscustomobject]@{
        BackupPath   = $backupPath
        ManifestPath = $manifestPath
        SizeBytes    = $file.Length
        SHA256       = $hostHash
    } | Format-List
}
finally {
    try {
        Invoke-Compose -Arguments @("exec", "-T", "mysql", "rm", "-f", $containerPath) | Out-Null
    }
    catch {
        Write-Warning "failed to remove temporary container backup: $($_.Exception.Message)"
    }
}
