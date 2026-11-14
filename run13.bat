@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo == apply phase12 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase12
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
if not exist eval\ubaps.edges.json (
  echo == first run: sampling edges for you to label
  pidx.exe eval -root "%TARGET%" sample out=eval\ubaps.edges.json n=40 mode=linked
  pidx.exe eval -root "%TARGET%" sample out=eval\ubaps.edges.json n=25 mode=unlinked
)
echo == task evaluation
pidx.exe eval -root "%TARGET%" tasks file=eval\ubaps.tasks.json
echo == edge evaluation (needs reviewed labels)
pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
echo == regression gate
if not exist eval\ubaps.baseline.json (
  pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json update=1
) else (
  pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
)
echo == done
