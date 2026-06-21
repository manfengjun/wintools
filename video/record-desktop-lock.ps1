param(
    [string]$DemoExe = (Join-Path $PSScriptRoot '..\build\bin\Wintools-demo.exe'),
    [string]$OutputDir = 'D:\JJGIT\ai-workspace\artifacts\wintools-douyin\01-desktop-lock',
    [string]$DemoPassword = $env:WINTOOLS_DEMO_PASSWORD
)

$ErrorActionPreference = 'Stop'

if (-not ('DemoWindow' -as [type])) {
    Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class DemoWindow {
    [DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
}
'@
}

function Show-DemoWindow {
    $process = Get-Process Wintools-demo -ErrorAction Stop | Select-Object -First 1
    if ($process.MainWindowHandle -eq 0) { throw 'Wintools demo window is unavailable.' }
    [DemoWindow]::ShowWindowAsync($process.MainWindowHandle, 3) | Out-Null
    [DemoWindow]::SetForegroundWindow($process.MainWindowHandle) | Out-Null
    [DemoWindow]::SetCursorPos(1100, 300) | Out-Null
    Start-Sleep -Seconds 1
}

function Minimize-OtherWindows {
    $handles = @()
    Get-Process | Where-Object {
        $_.MainWindowHandle -ne 0 -and
        $_.ProcessName -notin @('Wintools', 'Wintools-demo') -and
        -not [DemoWindow]::IsIconic($_.MainWindowHandle)
    } | ForEach-Object {
        $handles += $_.MainWindowHandle
        [DemoWindow]::ShowWindowAsync($_.MainWindowHandle, 6) | Out-Null
    }
    return $handles
}

function Find-FFmpeg {
    $command = Get-Command ffmpeg.exe -ErrorAction SilentlyContinue
    if ($command) { return $command.Source }

    $packageRoot = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages\Gyan.FFmpeg_Microsoft.Winget.Source_8wekyb3d8bbwe'
    $candidate = Get-ChildItem $packageRoot -Recurse -Filter ffmpeg.exe -ErrorAction SilentlyContinue |
        Select-Object -First 1 -ExpandProperty FullName
    if (-not $candidate) { throw 'FFmpeg is not installed.' }
    return $candidate
}

if (-not $DemoPassword) { throw 'WINTOOLS_DEMO_PASSWORD is required.' }
if (-not (Test-Path -LiteralPath $DemoExe)) { throw "Demo executable not found: $DemoExe" }

$ffmpeg = Find-FFmpeg
$desktop = [Environment]::GetFolderPath('Desktop')
$shortcut = Join-Path $desktop 'Wintools演示.lnk'
$backup = Join-Path $env:APPDATA 'DesktopSuite\lock-backup\Wintools演示.lnk'
$rawVideo = Join-Path $OutputDir 'screen-raw.mkv'

if (Test-Path -LiteralPath $shortcut) { throw "Safety stop: demo shortcut already exists: $shortcut" }
if (Test-Path -LiteralPath $backup) { throw "Safety stop: demo backup already exists: $backup" }

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
Remove-Item -LiteralPath $rawVideo -ErrorAction SilentlyContinue

$recorder = $null
$minimizedHandles = @()
try {
    Get-Process Wintools,Wintools-demo -ErrorAction SilentlyContinue | Stop-Process -Force
    $minimizedHandles = @(Minimize-OtherWindows)

    $shell = New-Object -ComObject WScript.Shell
    $link = $shell.CreateShortcut($shortcut)
    $link.TargetPath = "$env:WINDIR\System32\notepad.exe"
    $link.Description = 'Wintools桌面锁自动录制测试'
    $link.Save()

    $env:WINTOOLS_DEMO_PASSWORD = $DemoPassword
    Start-Process -FilePath $DemoExe
    Start-Sleep -Seconds 5
    Show-DemoWindow

    $recorderArgs = @(
        '-y', '-f', 'gdigrab', '-framerate', '30', '-draw_mouse', '1',
        '-i', 'desktop', '-t', '28', '-an', '-c:v', 'libx264',
        '-preset', 'ultrafast', '-crf', '20', '-pix_fmt', 'yuv420p', $rawVideo
    )
    $recorder = Start-Process -FilePath $ffmpeg -ArgumentList $recorderArgs -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 2

    Start-Process -FilePath $DemoExe -ArgumentList '--demo-command=lock' -Wait
    Start-Sleep -Seconds 4
    if (-not (Test-Path -LiteralPath $backup)) { throw 'Desktop lock did not create the demo backup.' }

    Start-Process -FilePath $DemoExe -ArgumentList '--demo-command=minimize' -Wait
    [DemoWindow]::SetCursorPos(1100, 300) | Out-Null
    Start-Sleep -Seconds 2
    Remove-Item -LiteralPath $shortcut

    $restored = $false
    for ($i = 0; $i -lt 12; $i++) {
        Start-Sleep -Milliseconds 500
        if (Test-Path -LiteralPath $shortcut) { $restored = $true; break }
    }
    if (-not $restored) { throw 'Desktop lock did not restore the demo shortcut.' }
    Start-Sleep -Seconds 4

    Start-Process -FilePath $DemoExe -ArgumentList '--demo-command=show' -Wait
    Start-Sleep -Seconds 2
    Show-DemoWindow
    Start-Process -FilePath $DemoExe -ArgumentList '--demo-command=unlock' -Wait
    Start-Sleep -Seconds 5

    $recorder.WaitForExit()
    if ($recorder.ExitCode -ne 0) { throw "FFmpeg recording failed with exit code $($recorder.ExitCode)." }
}
finally {
    if ($recorder -and -not $recorder.HasExited) { $recorder.Kill() }
    if (Test-Path -LiteralPath $DemoExe) {
        Start-Process -FilePath $DemoExe -ArgumentList '--demo-command=unlock' -Wait -ErrorAction SilentlyContinue
    }
    Remove-Item -LiteralPath $shortcut -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $backup -ErrorAction SilentlyContinue
    Get-Process Wintools,Wintools-demo -ErrorAction SilentlyContinue | Stop-Process -Force
    foreach ($handle in $minimizedHandles) {
        [DemoWindow]::ShowWindowAsync([IntPtr]$handle, 9) | Out-Null
    }
    Remove-Item Env:WINTOOLS_DEMO_PASSWORD -ErrorAction SilentlyContinue
}

Get-Item -LiteralPath $rawVideo
