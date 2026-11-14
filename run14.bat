@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo == apply phase13 patches
go run ./tools/applypatch -root . patches/phase13
if errorlevel 1 exit /b 1
echo == go mod tidy
go mod tidy
if errorlevel 1 exit /b 1
echo == build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo == test
go test ./...
if errorlevel 1 exit /b 1
echo == scan target
pidx.exe scan -root "%TARGET%"
echo.
echo ===== WHY: forgot_password =====
pidx.exe eval -root "%TARGET%" why file=eval\ubaps.tasks.json name=forgot_password
echo ===== WHY: admin_delete_user =====
pidx.exe eval -root "%TARGET%" why file=eval\ubaps.tasks.json name=admin_delete_user
if not exist eval\decisions.txt goto sheet
echo == applying your decisions from eval\decisions.txt
pidx.exe eval -root "%TARGET%" apply labels=eval\ubaps.edges.json decisions=eval\decisions.txt
echo == edge evaluation
pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
echo == regression gate
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
:sheet
echo == review sheet for the labels still pending
pidx.exe eval -root "%TARGET%" sheet labels=eval\ubaps.edges.json out=eval\review.txt
echo.
echo NEXT: open eval\review.txt, write one decision per line in eval\decisions.txt, then run this file again
echo == done
