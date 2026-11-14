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
go run ./tools/applypatch -root . patches/phase19
if errorlevel 1 exit /b 1
echo [%time%] build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo [%time%] test
go test ./...
if errorlevel 1 exit /b 1
echo [%time%] scan target (extractor version bump: full re-extract)
pidx.exe scan -root "%TARGET%"
echo [%time%] apply decisions
pidx.exe eval -root "%TARGET%" apply labels=eval\ubaps.edges.json decisions=eval\decisions.txt
echo [%time%] edge evaluation
pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
echo [%time%] gate against the old baseline (expect: tasks.noise back to 0, 6 edge metrics new)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
echo [%time%] accept the new baseline (task metrics plus edge metrics)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json update=1
echo [%time%] verify
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
echo [%time%] unresolved edges now (patch_reports.py noise should be gone)
pidx.exe unresolved -root "%TARGET%" 40
echo [%time%] done
