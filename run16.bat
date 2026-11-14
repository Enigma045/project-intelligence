@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo [%time%] build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo [%time%] test
go test ./...
if errorlevel 1 exit /b 1
echo [%time%] scan target
pidx.exe scan -root "%TARGET%"
echo [%time%] accept new baseline (task metrics; edge metrics join once labels are reviewed)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json update=1
echo [%time%] verify gate against the new baseline
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
if exist eval\decisions.txt (
  echo [%time%] applying your decisions
  pidx.exe eval -root "%TARGET%" apply labels=eval\ubaps.edges.json decisions=eval\decisions.txt
  pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
)
echo [%time%] done
