param()

# 测试 1: 检查 Makefile 是否存在
if (-Not (Test-Path "Makefile")) {
    Write-Error "RED: Makefile not found"
    exit 1
} else {
    Write-Host "PASS: Makefile exists" -ForegroundColor Green
}

# 测试 2: 检查 build.ps1 是否存在
if (-Not (Test-Path "build.ps1")) {
    Write-Error "RED: build.ps1 not found"
    exit 1
} else {
    Write-Host "PASS: build.ps1 exists" -ForegroundColor Green
}

Write-Host "ALL TESTS PASSED" -ForegroundColor Green
exit 0
