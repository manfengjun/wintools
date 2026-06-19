<#
.SYNOPSIS
    码力工坊 一键构建发布脚本
.DESCRIPTION
    - build        : 本地构建 (wails build)
    - bump <type>  : 升级版本号 (major/minor/patch)
    - tag <ver>    : 打标签并推送 (git tag v1.0.x && git push)
    - release <ver>: 发布到 Gitee + GitHub Releases
    - all <type>   : bump → build → tag → release 全自动

    示例:
        .\build.ps1 build          # 仅本地构建
        .\build.ps1 bump patch     # 版本号 v1.0.0 → v1.0.1
        .\build.ps1 release v1.0.1 # 构建 + 发布 Release
        .\build.ps1 all patch      # 全自动：bump→build→tag→release
#>

param(
    [ValidateSet('build','bump','tag','release','all')]
    [string]$Command = 'build',
    [string]$Argument = ''
)

$ErrorActionPreference = 'Stop'
$ProjectDir = Split-Path $PSScriptRoot -Parent

# ── 版本号管理 ──────────────────────────────────────────────

$VersionFile = "$ProjectDir\internal\updater\checker.go"

function Get-CurrentVersion {
    $content = Get-Content $VersionFile -Raw
    if ($content -match 'CurrentVersion\s*=\s*"([^"]+)"') {
        return $matches[1]
    }
    throw "无法从 checker.go 读取版本号"
}

function Bump-Version {
    param([string]$BumpType)
    $ver = Get-CurrentVersion
    $parts = $ver.Split('.')
    $major = [int]$parts[0]
    $minor = [int]$parts[1]
    $patch = [int]$parts[2]

    switch ($BumpType) {
        'major' { $major++; $minor = 0; $patch = 0 }
        'minor' { $minor++; $patch = 0 }
        'patch' { $patch++ }
        default { throw "bump 类型必须是 major/minor/patch" }
    }
    $newVer = "$major.$minor.$patch"
    Write-Host "🔖 $ver → $newVer" -ForegroundColor Cyan

    # 更新 checker.go
    $content = Get-Content $VersionFile -Raw
    $content = $content -replace 'CurrentVersion\s*=\s*"[^"]+"', "CurrentVersion = `"$newVer`""
    Set-Content $VersionFile -Value $content -NoNewline

    # 更新 README 中的版本引用 (不做精确匹配，仅提示)
    Write-Host "  已更新 internal/updater/checker.go" -ForegroundColor Green
    return $newVer
}

# ── 构建 ────────────────────────────────────────────────────

function Invoke-Build {
    Write-Host "🔨 构建中..." -ForegroundColor Cyan
    $ver = Get-CurrentVersion
    Push-Location $ProjectDir
    try {
        wails build
        Write-Host "✅ 构建成功: $ver" -ForegroundColor Green
        $exe = "build\bin\Wintools.exe"
        if (Test-Path $exe) {
            $size = (Get-Item $exe).Length / 1MB
            Write-Host "   输出: $exe ({0:N1} MB)" -f $size -ForegroundColor Green
        }
    } finally {
        Pop-Location
    }
}

# ── Git Tag ─────────────────────────────────────────────────

function Invoke-Tag {
    param([string]$Version)
    if (-not $Version) { $Version = Get-CurrentVersion }
    $tag = "v$Version"

    Push-Location $ProjectDir
    try {
        git add -A
        git commit -m "release $tag" --allow-empty
        git tag -d $tag 2>$null
        git tag $tag
        Write-Host "🏷️  Tag $tag 已创建" -ForegroundColor Cyan

        # 推送到所有 remote
        git remote | ForEach-Object {
            try {
                git push $_ master 2>&1 | Out-Null
                git push $_ $tag 2>&1 | Out-Null
                Write-Host "   ✅ 推送到 $_" -ForegroundColor Green
            } catch {
                Write-Warning "   ⚠️ 推送 $_ 失败: $_"
            }
        }
    } finally {
        Pop-Location
    }
}

# ── Release ─────────────────────────────────────────────────

function New-Release {
    param([string]$Version)
    if (-not $Version) { $Version = Get-CurrentVersion }
    $tag = "v$Version"
    $exe = "$ProjectDir\build\bin\Wintools.exe"

    if (-not (Test-Path $exe)) {
        Write-Host "⚠️ 未找到构建产物，先运行 build" -ForegroundColor Yellow
        Invoke-Build
    }

    Write-Host "📦 发布 $tag ..." -ForegroundColor Cyan
    $notes = @"
更新说明:
- 请在此处填写更新内容
- 例如: 修复 xxx 问题
"@

    # 发布到 Gitee
    try {
        gh release create $tag "$exe#Wintools_Windows_x86_64.exe" `
            --title "Wintools $tag" --notes $notes `
            -R gitee.com/manfengjun/wintools 2>&1 | Out-Null
        Write-Host "   ✅ Gitee release 已创建" -ForegroundColor Green
    } catch {
        Write-Warning "   ⚠️ Gitee 发布失败: $_"
    }

    # 发布到 GitHub
    try {
        gh release create $tag "$exe#Wintools_Windows_x86_64.exe" `
            --title "Wintools $tag" --notes $notes `
            -R github.com/manfengjun/wintools 2>&1 | Out-Null
        Write-Host "   ✅ GitHub release 已创建" -ForegroundColor Green
    } catch {
        Write-Warning "   ⚠️ GitHub 发布失败: $_"
    }

    Write-Host "✅ 发布完成: $tag" -ForegroundColor Green
}

# ── 主流程 ──────────────────────────────────────────────────

switch ($Command) {
    'build' {
        Invoke-Build
    }
    'bump' {
        if (-not $Argument) { $Argument = 'patch' }
        Bump-Version $Argument
    }
    'tag' {
        Invoke-Tag $Argument
    }
    'release' {
        if (-not $Argument) { $Argument = Get-CurrentVersion }
        Invoke-Build
        New-Release $Argument
    }
    'all' {
        if (-not $Argument) { $Argument = 'patch' }
        $newVer = Bump-Version $Argument
        Invoke-Build
        Invoke-Tag $newVer
        New-Release $newVer
    }
}
