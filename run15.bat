@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo == apply phase14 patches
go run ./tools/applypatch -root . patches/phase14
if errorlevel 1 exit /b 1
echo == build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo == test
go test ./...
if errorlevel 1 exit /b 1
echo == scan target
pidx.exe scan -root "%TARGET%"
echo == task evaluation
pidx.exe eval -root "%TARGET%" tasks file=eval\ubaps.tasks.json
echo == why: forgot_password
pidx.exe eval -root "%TARGET%" why file=eval\ubaps.tasks.json name=forgot_password
echo == regression gate (baseline NOT updated; improvements show as better)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
if exist eval\decisions.txt (
  echo == applying your decisions
  pidx.exe eval -root "%TARGET%" apply labels=eval\ubaps.edges.json decisions=eval\decisions.txt
  pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
)
pidx.exe eval -root "%TARGET%" sheet labels=eval\ubaps.edges.json out=eval\review.txt
echo == done
