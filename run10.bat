@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo == apply phase9 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase9
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
echo == scan target (extractor version bump rebuilds the graph)
pidx.exe scan -root "%TARGET%"
echo == scan again (expect all unchanged)
pidx.exe scan -root "%TARGET%"
echo == graph-stats
pidx.exe graph-stats -root "%TARGET%"
echo == routes written to routes.txt
pidx.exe routes -root "%TARGET%" > routes.txt
echo handlers still dynamic:
findstr /C:"handler: (dynamic)" routes.txt
echo handler counts by status:
findstr /C:"handler:" routes.txt | find /c "[resolved"
findstr /C:"handler:" routes.txt | find /c "[inferred"
findstr /C:"handler:" routes.txt | find /c "[dynamic"
findstr /C:"handler:" routes.txt | find /c "[external"
findstr /C:"handler:" routes.txt | find /c "[unresolved"
echo handlers resolved through an alias variable or unresolved:
findstr /C:"handler: var:" /C:"[unresolved" routes.txt
echo == unresolved or ambiguous (first 30)
pidx.exe unresolved -root "%TARGET%" 30
echo == context: student application task
pidx.exe context -root "%TARGET%" "Student application form submission should validate input before saving"
echo == bench
pidx.exe bench -root "%TARGET%" "Student application form submission should validate input before saving"
echo == context: login task
pidx.exe context -root "%TARGET%" "Modify login to authorize users and redirect by role"
echo == done
