$docsPath = "d:\claudeprj\deploy\doc"

$coreClasses = @(
    "Deployer\Deployer",
    "Deployer\Host\Host",
    "Deployer\Task\Task",
    "Deployer\Collection\Collection"
)

Write-Host "--- 开始文档一致性审计 ---" -ForegroundColor Cyan

$docs = Get-ChildItem $docsPath -Filter "*.md"
if ($docs.Count -lt 3) {
    Write-Host "FAIL: 缺失文档文件" -ForegroundColor Red
} else {
    foreach ($doc in $docs) {
        $content = Get-Content $doc.FullName -Raw
        if ([string]::IsNullOrWhiteSpace($content)) {
            Write-Host "FAIL: 文档 $($doc.Name) 为空" -ForegroundColor Red
        } else {
            foreach ($class in $coreClasses) {
                if ($content -notmatch [regex]::Escape($class)) {
                    Write-Host "FAIL: 文档 $($doc.Name) 缺失类引用: $class" -ForegroundColor Red
                } else {
                    Write-Host "PASS: 文档 $($doc.Name) 包含 $class" -ForegroundColor Green
                }
            }
        }
    }
}
Write-Host "--- 审计结束 ---"
