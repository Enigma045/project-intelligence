@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo [%time%] apply patches
go run ./tools/applypatch -root . patches/phase18
if errorlevel 1 exit /b 1
echo [%time%] build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo [%time%] test
go test ./...
if errorlevel 1 exit /b 1
echo [%time%] scan target (extractor version bump: full re-extract)
pidx.exe scan -root "%TARGET%"
echo [%time%] hubs: caller counts of the audit-log helpers (decides the hub threshold)
pidx.exe hubs -root "%TARGET%" Create_
echo [%time%] hubs: the 30 most-called symbols overall
pidx.exe hubs -root "%TARGET%"
echo [%time%] edge evaluation (the two API_BASE_URL misses should be fixed)
pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
echo [%time%] task evaluation (unchanged from run19 expected)
pidx.exe eval -root "%TARGET%" tasks file=eval\ubaps.tasks.json
echo [%time%] gate (baseline NOT updated)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
echo [%time%] done
