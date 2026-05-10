$env:Path = "C:\mingw64\bin;" + $env:Path

# Build 64-bit
Write-Host "Building SysView_x64.exe ..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"
go build -ldflags="-s -w -H windowsgui" -o SysView_x64.exe .
if ($LASTEXITCODE -eq 0) {
    $size = [math]::Round((Get-Item SysView_x64.exe).Length / 1MB, 2)
    Write-Host "SysView_x64.exe built successfully ($size MB)"
} else {
    Write-Host "Failed to build SysView_x64.exe"
    exit 1
}

# Build 32-bit (cross-compile requires 32-bit gcc, may not be available)
Write-Host "Building SysView_x86.exe ..."
$env:GOARCH = "386"
$env:CGO_ENABLED = "1"
go build -ldflags="-s -w -H windowsgui" -o SysView_x86.exe .
if ($LASTEXITCODE -eq 0) {
    $size = [math]::Round((Get-Item SysView_x86.exe).Length / 1MB, 2)
    Write-Host "SysView_x86.exe built successfully ($size MB)"
} else {
    Write-Host "32-bit build failed (needs 32-bit MinGW). Falling back to CGO_ENABLED=0 with -tags=ci ..."
    $env:CGO_ENABLED = "0"
    go build -tags=ci -ldflags="-s -w -H windowsgui" -o SysView_x86.exe .
    if ($LASTEXITCODE -eq 0) {
        $size = [math]::Round((Get-Item SysView_x86.exe).Length / 1MB, 2)
        Write-Host "SysView_x86.exe built with software renderer ($size MB)"
    } else {
        Write-Host "Failed to build SysView_x86.exe"
        exit 1
    }
}

Write-Host "`nAll builds complete!"
Get-Item SysView_*.exe | Select-Object Name, @{N='Size(MB)';E={[math]::Round($_.Length/1MB,2)}}
