param(
    [string]$DemoExe = (Join-Path $PSScriptRoot '..\build\bin\Wintools-demo.exe'),
    [string]$OutputDir = 'D:\JJGIT\ai-workspace\artifacts\wintools-douyin\02-python-env',
    [string]$DemoCommand = 'python-demo',
    [int]$Duration = 25
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

$ffmpeg = Get-ChildItem (Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages\Gyan.FFmpeg_Microsoft.Winget.Source_8wekyb3d8bbwe') -Recurse -Filter ffmpeg.exe |
    Select-Object -First 1 -ExpandProperty FullName
if (-not $ffmpeg) { throw 'FFmpeg is not installed.' }
if (-not (Test-Path -LiteralPath $DemoExe)) { throw "Demo executable not found: $DemoExe" }

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$rawVideo = Join-Path $OutputDir 'screen-raw.mkv'
Remove-Item -LiteralPath $rawVideo -ErrorAction SilentlyContinue

$recorder = $null
$minimizedHandles = @()
try {
    Get-Process Wintools,Wintools-demo -ErrorAction SilentlyContinue | Stop-Process -Force
    $minimizedHandles = @(Minimize-OtherWindows)
    Start-Process -FilePath $DemoExe
    Start-Sleep -Seconds 5
    Show-DemoWindow

    $args = @('-y', '-f', 'gdigrab', '-framerate', '30', '-draw_mouse', '1', '-i', 'desktop',
        '-t', $Duration, '-an', '-c:v', 'libx264', '-preset', 'ultrafast', '-crf', '20', '-pix_fmt', 'yuv420p', $rawVideo)
    $recorder = Start-Process -FilePath $ffmpeg -ArgumentList $args -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 1
    Start-Process -FilePath $DemoExe -ArgumentList "--demo-command=$DemoCommand" -Wait
    Start-Sleep -Seconds 8
    Show-DemoWindow
    $recorder.WaitForExit()
    if ($recorder.ExitCode -ne 0) { throw "FFmpeg recording failed with exit code $($recorder.ExitCode)." }
}
finally {
    if ($recorder -and -not $recorder.HasExited) { $recorder.Kill() }
    Get-Process Wintools,Wintools-demo -ErrorAction SilentlyContinue | Stop-Process -Force
    foreach ($handle in $minimizedHandles) { [DemoWindow]::ShowWindowAsync([IntPtr]$handle, 9) | Out-Null }
}

Get-Item -LiteralPath $rawVideo
