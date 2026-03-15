$ErrorActionPreference = 'Stop'

adb shell run-as com.sober.admin rm -f cache/sober_apps.json
adb shell am broadcast -a com.sober.LIST_APPS -n com.sober.admin/.CommandReceiver

$timeout = 5
$elapsed = 0
$result = $null

while ($elapsed -lt $timeout) {
    $result = adb shell run-as com.sober.admin cat cache/sober_apps.json 2>$null
    if ($result -match '^\[' -or $result -match '^\{"error"') {
        break
    }
    Start-Sleep -Milliseconds 250
    $elapsed++
}

if (-not $result) {
    Write-Error "Timed out waiting for results"
    exit 1
}

$outFile = "$PSScriptRoot\list-apps-output.json"

if (Get-Command jq -ErrorAction SilentlyContinue) {
    $result | jq . | Set-Content $outFile
} else {
    $result | Set-Content $outFile
}

Write-Output "Output saved to: $outFile"
