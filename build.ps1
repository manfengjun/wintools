<#
.SYNOPSIS
    码力工坊 一键构建发布脚本
.DESCRIPTION
    - build        : 构建 NSIS 安装包（内嵌 WebView2 联网引导程序）
    - bump <type>  : 升级版本号 (major/minor/patch)
    - tag <ver>    : 打标签并推送 (git tag v1.0.x && git push)
    - release <ver>: 发布到 Gitee + GitHub Releases
    - all <type>   : bump → build → tag → release 全自动

    示例:
        .\build.ps1 build          # 仅本地构建
        .\build.ps1 bump patch     # 版本号 v1.0.0 → v1.0.1
        .\build.ps1 release v1.0.1 # 构建 + 发布 Release
        .\build.ps1 all patch      # 全自动：bump→build→tag→release

    环境变量:
        GITEE_TOKEN    [必需] 用于发布到 Gitee Releases
        GITHUB_TOKEN   [可选] gh CLI 自动使用已登录的 GitHub CLI 会话
#>

param(
    [ValidateSet('build','bump','tag','release','all')]
    [string]$Command = 'build',
    [string]$Argument = ''
)

$ErrorActionPreference = 'Stop'
$ProjectDir = $PSScriptRoot

# ── Gitee Token ──────────────────────────────────────────
# 必须通过环境变量 GITEE_TOKEN 提供
$GiteeToken = $env:GITEE_TOKEN
$GiteeOwner = "3672830"
$GiteeRepo = "wintools"

# 获取当前仓库的默认分支名（支持 main / master / 其他）
function Get-DefaultBranch {
    $remote = git remote 2>$null | Select-Object -First 1
    if (-not $remote) { return "main" }
    $head = git remote show $remote 2>$null | Select-String "HEAD branch:" | ForEach-Object {
        $_.ToString().Split(":")[1].Trim()
    }
    if (-not $head) { $head = "main" }
    return $head
}

# ── 版本号管理 ──────────────────────────────────────────

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
    # 补齐到 3 段，避免版本号格式缩水
    while ($parts.Count -lt 3) { $parts += "0" }
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
    Set-Content $VersionFile -Value $content -Encoding UTF8

    Write-Host "  已更新 internal/updater/checker.go" -ForegroundColor Green
    return $newVer
}

# ── 构建 ────────────────────────────────────────────────

function Get-InstallerArtifact {
    $installer = Get-ChildItem "$ProjectDir\build\bin" -Filter "*installer*.exe" -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
    if (-not $installer) {
        return $null
    }
    return $installer
}

function Initialize-NSIS {
    if (Get-Command makensis -ErrorAction SilentlyContinue) {
        return
    }

    $knownPaths = @(
            "$env:ProgramFiles\NSIS",
            "${env:ProgramFiles(x86)}\NSIS",
            "$env:LOCALAPPDATA\WintoolsBuildTools\NSIS\Bin"
        ) | Where-Object { $_ -and (Test-Path (Join-Path $_ 'makensis.exe')) }

    if ($knownPaths.Count -gt 0) {
        $env:PATH = "$($knownPaths[0]);$env:PATH"
        return
    }

    throw "缺少 NSIS（makensis）。请先安装 NSIS 并重新运行；下载地址: https://nsis.sourceforge.io/Download"
}

function Invoke-Build {
    Write-Host "🔨 构建 NSIS 安装包（WebView2 在线引导）..." -ForegroundColor Cyan
    $ver = Get-CurrentVersion
    Initialize-NSIS
    Push-Location $ProjectDir
    try {
        # 清理前次构建的安装包
        Remove-Item -Path "$ProjectDir\build\bin\*installer*.exe" -Force -ErrorAction SilentlyContinue

        wails build -nsis -webview2 embed
        if ($LASTEXITCODE -ne 0) {
            throw "wails build 失败，退出码: $LASTEXITCODE"
        }

        $exe = Get-Item "build\bin\Wintools.exe" -ErrorAction Stop
        $installer = Get-InstallerArtifact
        if (-not $installer) {
            throw "未找到 NSIS 安装包（build\bin\*installer*.exe）"
        }

        Write-Host "✅ 构建成功: $ver" -ForegroundColor Green
        Write-Host ("   调试程序: {0} ({1:N1} MB)" -f $exe.FullName, ($exe.Length / 1MB)) -ForegroundColor DarkGreen
        Write-Host ("   发布安装包: {0} ({1:N1} MB)" -f $installer.FullName, ($installer.Length / 1MB)) -ForegroundColor Green
        return $installer.FullName
    } finally {
        Pop-Location
    }
}

# ── Git Tag ─────────────────────────────────────────────

function Invoke-Tag {
    param([string]$Version)
    if (-not $Version) { $Version = Get-CurrentVersion }
    $tag = "v$Version"
    $defaultBranch = Get-DefaultBranch

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
                git push $_ $defaultBranch 2>&1 | Out-Null
                git push $_ $tag 2>&1 | Out-Null
                Write-Host "   ✅ 推送到 $_ ($defaultBranch)" -ForegroundColor Green
            } catch {
                Write-Warning "   ⚠️ 推送 $_ 失败: $_"
            }
        }
    } finally {
        Pop-Location
    }
}

# ── Release ─────────────────────────────────────────────

function Get-ReleaseNotes {
    <#
    .SYNOPSIS
        从 git log 自动生成 Release notes。
        从上个 tag 到 HEAD 收集 commit 消息；无 tag 时返回初始版本提示。
    #>
    $lastTag = git describe --tags --abbrev=0 2>$null
    if ($lastTag) {
        $log = git log "$lastTag..HEAD" --format="* %s" 2>$null
        if ($log) {
            return "## 更新内容`n`n$log"
        }
    }
    return "## 更新内容`n`n- 各项改进与修复"
}

function New-Release {
    param([string]$Version)
    if (-not $Version) { $Version = Get-CurrentVersion }
    $tag = "v$Version"
    $installer = Get-InstallerArtifact

    if (-not $installer) {
        Write-Host "⚠️ 未找到 NSIS 安装包，先运行 build" -ForegroundColor Yellow
        Invoke-Build | Out-Null
        $installer = Get-InstallerArtifact
    }
    if (-not $installer) {
        throw "构建完成后仍未找到 NSIS 安装包"
    }

    Write-Host "📦 发布 $tag ..." -ForegroundColor Cyan
    $notes = Get-ReleaseNotes

    # 发布到 GitHub
    try {
        gh release create $tag "$($installer.FullName)#Wintools_Windows_x86_64_Setup.exe" `
            --title "Wintools $tag" --notes $notes `
            -R github.com/manfengjun/wintools 2>&1 | Out-Null
        Write-Host "   ✅ GitHub release 已创建" -ForegroundColor Green
    } catch {
        Write-Warning "   ⚠️ GitHub 发布失败: $_"
    }

    # 发布到 Gitee（需要 GITEE_TOKEN）
    try {
        if (-not $GiteeToken) { throw "环境变量 GITEE_TOKEN 未设置，请在运行前设置: `$env:GITEE_TOKEN = 'your_token'" }

        $bodyText = $notes -replace "`n", "`n"
        $payload = @{tag_name=$tag; name="Wintools $tag"; target_commitish=(Get-DefaultBranch); body=$bodyText} | ConvertTo-Json
        $uri = "https://gitee.com/api/v5/repos/$GiteeOwner/$GiteeRepo/releases?access_token=$GiteeToken"
        $result = Invoke-RestMethod $uri -Method Post -ContentType "application/json" -Body ([Text.Encoding]::UTF8.GetBytes($payload))

        # 上传附件
        $tmpExe = "$env:TEMP\Wintools_Windows_x86_64_Setup.exe"
        Copy-Item $installer.FullName $tmpExe -Force
        $uploadUri = "https://gitee.com/api/v5/repos/$GiteeOwner/$GiteeRepo/releases/$($result.id)/attach_files?access_token=$GiteeToken"
        $wc = New-Object System.Net.WebClient
        $wc.UploadFile($uploadUri, $tmpExe) | Out-Null
        Write-Host "   ✅ Gitee release 已创建" -ForegroundColor Green
    } catch {
        Write-Warning "   ⚠️ Gitee 发布失败: $_"
    }

    Write-Host "✅ 发布完成: $tag" -ForegroundColor Green
}

# ── 主流程 ──────────────────────────────────────────────

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
