@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo [%time%] call edges BEFORE
pidx.exe graph-stats -root "%TARGET%" | findstr /C:"calls/"
echo [%time%] apply patches
go run ./tools/applypatch -root . patches/phase20
if errorlevel 1 exit /b 1
echo [%time%] build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo [%time%] test
go test ./...
if errorlevel 1 exit /b 1
echo [%time%] scan target (extractor version bump: full re-extract)
pidx.exe scan -root "%TARGET%"
echo [%time%] call edges AFTER (calls/value should drop; resolved and inferred rise)
pidx.exe graph-stats -root "%TARGET%" | findstr /C:"calls/"
echo [%time%] gate against the baseline (NOT updated)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
echo [%time%] who references Deans_Decisions.js?
findstr /S /N /C:"Deans_Decisions.js" "%TARGET%\Pages\*.html" || echo (no html page loads it)
echo [%time%] sampling 30 Go call links (precision) and 20 remaining Go value edges (recall)
pidx.exe eval -root "%TARGET%" sample out=eval\ubaps.edges.json n=30 mode=linked lang=go types=calls seed=3
pidx.exe eval -root "%TARGET%" sample out=eval\ubaps.edges.json n=20 mode=value lang=go types=calls seed=3
pidx.exe eval -root "%TARGET%" sheet labels=eval\ubaps.edges.json out=eval\review.txt
echo [%time%] done
