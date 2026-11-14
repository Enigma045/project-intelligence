@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
if exist "..\Ubaps_Go-main\" set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found in project-intelligence or its parent folder
dir /b
exit /b 1
:found
echo == target: %TARGET%
echo == build pidx
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo == scan 1 (full index)
pidx.exe scan -root "%TARGET%"
echo == scan 2 (expect all unchanged)
pidx.exe scan -root "%TARGET%"
echo == stats
pidx.exe stats -root "%TARGET%"
echo == graph-stats
pidx.exe graph-stats -root "%TARGET%"
echo == generate progress.md and module shards
pidx.exe progress -root "%TARGET%"
echo == progress.md
type "%TARGET%\progress.md"
echo == routes (handlers, queries, JS callers)
pidx.exe routes -root "%TARGET%"
echo == database tables
pidx.exe db -root "%TARGET%"
echo == unresolved or ambiguous (first 40)
pidx.exe unresolved -root "%TARGET%" 40
echo == dynamic edges (first 15)
pidx.exe unresolved -root "%TARGET%" 15 dynamic
echo == done
