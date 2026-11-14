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
go run ./tools/applypatch -root . patches/phase17
if errorlevel 1 exit /b 1
echo [%time%] build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo [%time%] test
go test ./...
if errorlevel 1 exit /b 1
echo [%time%] scan target
pidx.exe scan -root "%TARGET%"
echo [%time%] task evaluation (expect noise 0 and student_application 5/5 again)
pidx.exe eval -root "%TARGET%" tasks file=eval\ubaps.tasks.json
echo [%time%] why: student_application
pidx.exe eval -root "%TARGET%" why file=eval\ubaps.tasks.json name=student_application
echo [%time%] apply decisions (44 48 45 49 updated)
pidx.exe eval -root "%TARGET%" apply labels=eval\ubaps.edges.json decisions=eval\decisions.txt
echo [%time%] edge evaluation (45 and 49 should show as misses: constants are not resolved yet)
pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
echo [%time%] gate (baseline NOT updated)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
echo [%time%] done
