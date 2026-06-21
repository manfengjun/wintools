param(
    [string]$OutputDir = 'D:\JJGIT\ai-workspace\artifacts\wintools-douyin\01-desktop-lock',
    [string]$FilterFile = 'render-filter.txt',
    [string]$OutputFile = '01-wintools-desktop-lock-douyin.mp4',
    [int]$Duration = 22
)

$ErrorActionPreference = 'Stop'
$ffmpeg = Get-ChildItem (Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages\Gyan.FFmpeg_Microsoft.Winget.Source_8wekyb3d8bbwe') -Recurse -Filter ffmpeg.exe |
    Select-Object -First 1 -ExpandProperty FullName
if (-not $ffmpeg) { throw 'FFmpeg is not installed.' }

$filter = Join-Path $PSScriptRoot $FilterFile
$screen = Join-Path $OutputDir 'screen-raw.mkv'
$voice = Join-Path $OutputDir 'narration.mp3'
$subtitles = Join-Path $OutputDir 'narration.srt'
$output = Join-Path $OutputDir $OutputFile

foreach ($path in @($filter, $screen, $voice, $subtitles)) {
    if (-not (Test-Path -LiteralPath $path)) { throw "Required render input is missing: $path" }
}

Copy-Item -LiteralPath $subtitles -Destination (Join-Path $PSScriptRoot 'narration.srt') -Force
try {
    Push-Location $PSScriptRoot
    & $ffmpeg -y `
        -f lavfi -i "color=c=0xF3F0FF:s=1080x1920:r=30:d=$Duration" `
        -i $screen -i $voice `
        -filter_complex_script $filter `
        -map '[v]' -map '2:a:0' -t $Duration `
        -c:v libx264 -preset medium -crf 18 -pix_fmt yuv420p `
        -c:a aac -b:a 192k -movflags +faststart $output
    if ($LASTEXITCODE -ne 0) { throw "FFmpeg render failed with exit code $LASTEXITCODE." }
}
finally {
    Pop-Location
    Remove-Item -LiteralPath (Join-Path $PSScriptRoot 'narration.srt') -ErrorAction SilentlyContinue
}

Get-Item -LiteralPath $output
