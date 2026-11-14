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
go run ./tools/applypatch -root . patches/phase15
if errorlevel 1 exit /b 1
echo [%time%] build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo [%time%] test
go test ./...
if errorlevel 1 exit /b 1
echo [%time%] scan target
pidx.exe scan -root "%TARGET%"
echo [%time%] apply decisions
pidx.exe eval -root "%TARGET%" apply labels=eval\ubaps.edges.json decisions=eval\decisions.txt
echo [%time%] edge evaluation
pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
echo [%time%] gate (baseline NOT updated: new edge metrics are listed as new, not as failures)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
echo [%time%] done
