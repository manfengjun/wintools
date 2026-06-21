param(
    [string]$OutputDir = 'D:\JJGIT\ai-workspace\artifacts\wintools-douyin\04-open-source',
    [string]$FilterFile = 'render-filter-open-source.txt',
    [string]$OutputFile = '04-wintools-open-source-douyin.mp4',
    [int]$Duration = 23
)

$ErrorActionPreference = 'Stop'
$ffmpeg = Get-ChildItem (Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages\Gyan.FFmpeg_Microsoft.Winget.Source_8wekyb3d8bbwe') -Recurse -Filter ffmpeg.exe |
    Select-Object -First 1 -ExpandProperty FullName
if (-not $ffmpeg) { throw 'FFmpeg is not installed.' }

$filter = Join-Path $PSScriptRoot $FilterFile
$voice = Join-Path $OutputDir 'narration.mp3'
$output = Join-Path $OutputDir $OutputFile
foreach ($path in @($filter, $voice)) {
    if (-not (Test-Path -LiteralPath $path)) { throw "Required render input is missing: $path" }
}

& $ffmpeg -y `
    -f lavfi -i "color=c=0xF3F0FF:s=1080x1920:r=30:d=$Duration" `
    -i $voice -filter_complex_script $filter `
    -map '[v]' -map '1:a:0' -t $Duration `
    -c:v libx264 -preset medium -crf 18 -pix_fmt yuv420p `
    -c:a aac -b:a 192k -movflags +faststart $output
if ($LASTEXITCODE -ne 0) { throw "FFmpeg render failed with exit code $LASTEXITCODE." }

Get-Item -LiteralPath $output
