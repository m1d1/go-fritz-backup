#!/usr/bin/env pwsh

# Example file
# run go-fritz-backup
# cleanup older files > ROTATE_PERIOD

$ROTATE_PERIOD=14
$BACKUP_PATH="c:\tmp\backups"
$BINARY_PATH="c:\tmp"

try {
    Set-Location -Path "$BINARY_PATH"
    Invoke-Expression -Command "go-fritz-backup-windows-amd64.exe"    
    Get-ChildItem -PATH "$BACKUP_PATH" -ErrorAction SilentlyContinue | Where-Object { ((Get-Date)-$_.LastWriteTime).days -gt $ROTATE_PERIOD } | Remove-Item -Force
} catch {
    Write-Host $_.Exception.Message
    Exit(1)
}
