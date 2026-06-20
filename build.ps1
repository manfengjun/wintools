<#
.SYNOPSIS
    码力工坊 一键构建发布脚本
.DESCRIPTION
    - build        : 构建 Wintools.exe（有 NSIS 时同时生成安装包）
    - bump <type>  : 升级版本号 (major/minor/patch)
    - tag <ver>    : 打标签并推送 (git tag v1.0.x && git push)
    - release <ver>: 发布到 Gitee + GitHub Releases（自动上传构建产物）
    - all <type>   : bump → build → tag → release 全自动

    示例:
        .\build.ps1 build          # 仅本地构建
        .\build.ps1 bump patch     # 版本号 v1.0.0 → v1.0.1
        .\build.ps1 release v1.0.1 # 构建 + 发布 Release
        .\build.ps1 all patch      # 全自动：bump→build→tag→release

    Token 配置:
        在 local-tokens.ps1 中存放（已 .gitignore 排除，绝不提交）:
            $env:GITEE_TOKEN = "你的GiteeToken"   # https://gitee.com/profile/personal_access_tokens
            $env:GH_TOKEN    = "你的GitHubToken"  # https://github.com/settings/tokens (repo 权限)
        build.ps1 启动时自动加载该文件。

    无需 NSIS。有 NSIS 环境时自动生成安装包，无 NSIS 时仅生成 Wintools.exe。
#>

param(
    [ValidateSet('build','bump','tag','release','all')]
    [string]$Command = 'build',
    [string]$Argument = ''
)

$ErrorActionPreference = 'Stop'
$ProjectDir = $PSScriptRoot

# ── 本地 Token 配置（不提交到 git）────────────────────
$LocalTokens = Join-Path $ProjectDir "local-tokens.ps1"
if (Test-Path $LocalTokens) {
    . $LocalTokens
    Write-Host "  已加载 local-tokens.ps1" -ForegroundColor DarkGray
}

$GiteeOwner = "3672830"
$GiteeRepo = "wintools"
$AssetName = "Wintools_Windows_x86_64.exe"

# ── 工具函数 ──────────────────────────────────────────

# 获取当前仓库的默认分支名
function Get-DefaultBranch {
    $remote = git remote 2>$null | Select-Object -First 1
    if (-not $remote) { return "main" }
    $head = git remote show $remote 2>$null | Select-String "HEAD branch:" | ForEach-Object {
        $_.ToString().Split(":")[1].Trim()
    }
    if (-not $head) { $head = "main" }
    return $head
}

# 查找构建产物：优先 NSIS 安装包，其次 Wintools.exe
function Get-BuildArtifact {
    # 优先找 NSIS 安装包
    $installer = Get-ChildItem "$ProjectDir\build\bin" -Filter "*installer*.exe" -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($installer) { return $installer }

    # 退回到裸 EXE
    $exe = Get-ChildItem "$ProjectDir\build\bin\Wintools.exe" -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($exe) { return $exe }

    return $null
}

# ── 版本号管理 ────────────────────────────────────────

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

    $content = Get-Content $VersionFile -Raw
    $content = $content -replace 'CurrentVersion\s*=\s*"[^"]+"', "CurrentVersion = `"$newVer`""
    Set-Content $VersionFile -Value $content -Encoding UTF8
    Write-Host "  已更新 internal/updater/checker.go" -ForegroundColor Green
    return $newVer
}

# ── 构建 ──────────────────────────────────────────────

function Initialize-NSIS {
    # 有 NSIS 就用，没有就跳过（不报错）
    if (Get-Command makensis -ErrorAction SilentlyContinue) { return $true }

    $knownPaths = @(
            "$ProjectDir\build\nsis",
            "$env:ProgramFiles\NSIS",
            "${env:ProgramFiles(x86)}\NSIS",
            "$env:LOCALAPPDATA\WintoolsBuildTools\NSIS\Bin"
        ) | Where-Object { $_ -and (Test-Path (Join-Path $_ 'makensis.exe')) }

    if ($knownPaths) {
        $dir = @($knownPaths)[0]
        $env:PATH = "$dir;$env:PATH"
        # 确认 makensis 能真正运行
        $test = cmd /c "$dir\makensis.exe -VERSION" 2>$null
        if ($test) {
            Write-Host "  NSIS $($test.Trim()) 已就绪" -ForegroundColor DarkGray
            return $true
        }
    }

    Write-Host "  ⚠️ NSIS 未安装，跳过安装包生成" -ForegroundColor Yellow
    return $false
}

function Invoke-Build {
    Write-Host "🔨 构建 Wintools..." -ForegroundColor Cyan
    $ver = Get-CurrentVersion

    # 清理旧构建产物
    Remove-Item -Path "$ProjectDir\build\bin\*" -Force -ErrorAction SilentlyContinue

    $hasNSIS = Initialize-NSIS

    Push-Location $ProjectDir
    try {
        if ($hasNSIS) {
            wails build -nsis -webview2 embed
        } else {
            wails build
        }
        if ($LASTEXITCODE -ne 0) {
            throw "wails build 失败，退出码: $LASTEXITCODE"
        }

        $exe = Get-Item "build\bin\Wintools.exe" -ErrorAction Stop
        $artifact = Get-BuildArtifact

        Write-Host "✅ 构建成功: $ver" -ForegroundColor Green
        Write-Host ("   可执行文件: {0} ({1:N1} MB)" -f $exe.FullName, ($exe.Length / 1MB)) -ForegroundColor DarkGreen
        if ($artifact.FullName -ne $exe.FullName) {
            Write-Host ("   安装包: {0} ({1:N1} MB)" -f $artifact.FullName, ($artifact.Length / 1MB)) -ForegroundColor Green
        }
        return $artifact.FullName
    } finally {
        Pop-Location
    }
}

# ── Git Tag ───────────────────────────────────────────

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

# ── Release ───────────────────────────────────────────

function Get-ReleaseNotes {
    <#
    .SYNOPSIS
        从 git log 自动生成 Release notes。
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

function Publish-GitHub {
    param([string]$Tag, [string]$ArtifactPath, [string]$Notes)

    # 方式一：gh CLI（优先，用 --notes-file - 从 stdin 读多行 notes）
    try {
        $Notes | gh release create $Tag "$($ArtifactPath)#$AssetName" `
            --title "Wintools $Tag" --notes-file - `
            -R github.com/manfengjun/wintools 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "   ✅ GitHub release 已创建 (gh CLI)" -ForegroundColor Green
            return
        }
        Write-Host "   gh CLI 退出码 $LASTEXITCODE，尝试 API 直连..." -ForegroundColor DarkGray
    } catch {
        Write-Host "   gh CLI 不可用，尝试 API 直连..." -ForegroundColor DarkGray
    }

    # 方式二：GH_TOKEN API 直连
    $token = $env:GH_TOKEN
    if (-not $token) { throw "GH_TOKEN 未设置" }

    $headers = @{
        Authorization = "Bearer $token"
        Accept = "application/vnd.github.v3+json"
    }

    # 创建 Release
    $body = @{
        tag_name = $Tag
        name = "Wintools $Tag"
        body = $Notes
    } | ConvertTo-Json

    $result = Invoke-RestMethod -Uri "https://api.github.com/repos/manfengjun/wintools/releases" `
        -Method Post -Headers $headers -Body $body -ContentType "application/json"

    # 上传附件
    $uploadUri = "https://uploads.github.com/repos/manfengjun/wintools/releases/$($result.id)/assets?name=$AssetName"
    Invoke-RestMethod -Uri $uploadUri -Method Post -Headers $headers `
        -InFile $ArtifactPath -ContentType "application/x-msdownload" | Out-Null
    Write-Host "   ✅ GitHub release 已创建 (API)" -ForegroundColor Green
}

function Publish-Gitee {
    param([string]$Tag, [string]$ArtifactPath, [string]$Notes)

    $token = $env:GITEE_TOKEN
    if (-not $token) { throw "环境变量 GITEE_TOKEN 未设置" }

    # 创建 Release
    $body = @{
        tag_name = $Tag
        name = "Wintools $Tag"
        target_commitish = (Get-DefaultBranch)
        body = $Notes
    } | ConvertTo-Json

    $uri = "https://gitee.com/api/v5/repos/$GiteeOwner/$GiteeRepo/releases?access_token=$token"
    $result = Invoke-RestMethod $uri -Method Post -ContentType "application/json" -Body ([Text.Encoding]::UTF8.GetBytes($body))

    # 上传附件（用源文件名，Gitee 自动保留上传的文件名）
    $tmpFile = "$env:TEMP\$AssetName"
    Copy-Item $ArtifactPath $tmpFile -Force
    $uploadUri = "https://gitee.com/api/v5/repos/$GiteeOwner/$GiteeRepo/releases/$($result.id)/attach_files?access_token=$token"
    $wc = New-Object System.Net.WebClient
    $wc.UploadFile($uploadUri, $tmpFile) | Out-Null
    Write-Host "   ✅ Gitee release 已创建" -ForegroundColor Green
}

function New-Release {
    param([string]$Version)
    if (-not $Version) { $Version = Get-CurrentVersion }
    $tag = "v$Version"
    $notes = Get-ReleaseNotes

    # 确保构建产物存在
    $artifact = Get-BuildArtifact
    if (-not $artifact) {
        Write-Host "⚠️ 未找到构建产物，先运行 build" -ForegroundColor Yellow
        $null = Invoke-Build
        $artifact = Get-BuildArtifact
    }
    if (-not $artifact) {
        throw "构建完成后仍未找到产物"
    }

    Write-Host "📦 发布 $tag ..." -ForegroundColor Cyan
    Write-Host "   附件: $($artifact.Name) ($([math]::Round($artifact.Length/1MB, 1)) MB)" -ForegroundColor DarkGray

    # 发布到 GitHub
    try {
        Publish-GitHub -Tag $tag -ArtifactPath $artifact.FullName -Notes $notes
    } catch {
        Write-Warning "   ⚠️ GitHub 发布失败: $_"
    }

    # 发布到 Gitee
    try {
        Publish-Gitee -Tag $tag -ArtifactPath $artifact.FullName -Notes $notes
    } catch {
        Write-Warning "   ⚠️ Gitee 发布失败: $_"
    }

    Write-Host "✅ 发布完成: $tag" -ForegroundColor Green
}

# ── 主流程 ────────────────────────────────────────────

switch ($Command) {
    'build' { Invoke-Build }
    'bump' {
        if (-not $Argument) { $Argument = 'patch' }
        Bump-Version $Argument
    }
    'tag' { Invoke-Tag $Argument }
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
