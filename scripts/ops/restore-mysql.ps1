[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$BackupPath,
    [string]$ProjectDirectory = "",
    [string]$RestoreDatabase = "go_user_system_restore_test",
    [string]$ComposeProjectName = "",
    [string]$EvidenceDirectory = "",
    [switch]$ConfirmRestore
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if (-not $ConfirmRestore) {
    throw "-ConfirmRestore is required"
}
if ($RestoreDatabase -notmatch '^[A-Za-z0-9_]+_restore_test$') {
    throw "RestoreDatabase must contain only letters, numbers, underscores and end with _restore_test"
}

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

function Invoke-ComposeWithInput {
    param(
        [Parameter(Mandatory)]
        [string]$InputText,
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
        $output = $InputText | & docker @composeArguments @Arguments 2>&1
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
$BackupPath = (Resolve-Path $BackupPath).Path
$manifestPath = "$BackupPath.manifest.json"
if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    throw "backup manifest is required: $manifestPath"
}

$manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
if ($manifest.schema_version -ne 1 -or $manifest.database -ne "go_user_system") {
    throw "unsupported or invalid backup manifest"
}
$backupFile = Get-Item -LiteralPath $BackupPath
$backupHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $BackupPath).Hash.ToLowerInvariant()
if ($backupHash -ne $manifest.sha256.ToLowerInvariant() -or $backupFile.Length -ne $manifest.size_bytes) {
    throw "backup file does not match its manifest"
}

if ([string]::IsNullOrWhiteSpace($EvidenceDirectory)) {
    $EvidenceDirectory = Join-Path $ProjectDirectory "artifacts/operations"
}
$EvidenceDirectory = [System.IO.Path]::GetFullPath($EvidenceDirectory)
New-Item -ItemType Directory -Force $EvidenceDirectory | Out-Null

$runningServices = @(Invoke-Compose -Arguments @("ps", "--status", "running", "--services"))
if ($runningServices -notcontains "mysql") {
    throw "the Compose mysql service is not running"
}
$applicationUser = ((Invoke-Compose -Arguments @("exec", "-T", "mysql", "printenv", "MYSQL_USER")) | Select-Object -Last 1).Trim()
if ($applicationUser -notmatch '^[A-Za-z0-9_]+$') {
    throw "the Compose MYSQL_USER must contain only letters, numbers, and underscores"
}

$startedAt = (Get-Date).ToUniversalTime()
$containerPath = "/tmp/go_user_system-restore-$($startedAt.ToString('yyyyMMdd-HHmmss')).sql"
$mysqlArguments = @("exec", "-T", "mysql", "sh", "-c", 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot -N -B')
$createSql = "DROP DATABASE IF EXISTS $RestoreDatabase; CREATE DATABASE $RestoreDatabase CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; GRANT ALL PRIVILEGES ON $RestoreDatabase.* TO '$applicationUser'@'%';"
$restoreSql = "USE $RestoreDatabase; SOURCE $containerPath;"
$tableCountSql = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$RestoreDatabase';"
$migrationVersionSql = "USE $RestoreDatabase; SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1;"

try {
    Invoke-Compose -Arguments @("cp", $BackupPath, "mysql:$containerPath") | Out-Null
    Invoke-ComposeWithInput -InputText $createSql -Arguments $mysqlArguments | Out-Null
    Invoke-ComposeWithInput -InputText $restoreSql -Arguments $mysqlArguments | Out-Null

    $tableCount = ((Invoke-ComposeWithInput -InputText $tableCountSql -Arguments $mysqlArguments) | Select-Object -Last 1).Trim()
    $migrationVersion = ((Invoke-ComposeWithInput -InputText $migrationVersionSql -Arguments $mysqlArguments) | Select-Object -Last 1).Trim()
    if ([int]$tableCount -le 0 -or [int64]$migrationVersion -le 0) {
        throw "restored database validation failed: tables=$tableCount migration=$migrationVersion"
    }

    $completedAt = (Get-Date).ToUniversalTime()
    $evidence = [ordered]@{
        schema_version      = 1
        status              = "passed"
        source_backup       = $backupFile.Name
        source_sha256       = $backupHash
        restore_database    = $RestoreDatabase
        application_user    = $applicationUser
        started_at_utc      = $startedAt.ToString("o")
        completed_at_utc    = $completedAt.ToString("o")
        duration_seconds    = [math]::Round(($completedAt - $startedAt).TotalSeconds, 3)
        table_count         = [int]$tableCount
        migration_version   = [int64]$migrationVersion
    }
    $evidencePath = Join-Path $EvidenceDirectory "restore-drill-$($completedAt.ToString('yyyyMMdd-HHmmss')).json"
    $evidence | ConvertTo-Json | Set-Content -LiteralPath $evidencePath -Encoding utf8

    [pscustomobject]@{
        RestoreDatabase  = $RestoreDatabase
        TableCount       = [int]$tableCount
        MigrationVersion = [int64]$migrationVersion
        DurationSeconds  = $evidence.duration_seconds
        EvidencePath     = $evidencePath
    } | Format-List
}
finally {
    try {
        Invoke-Compose -Arguments @("exec", "-T", "mysql", "rm", "-f", $containerPath) | Out-Null
    }
    catch {
        Write-Warning "failed to remove temporary restore file: $($_.Exception.Message)"
    }
}
