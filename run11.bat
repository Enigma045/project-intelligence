@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo == apply phase10 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase10
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
echo == graph-stats (compare: calls/unresolved was 149, calls/value 855)
pidx.exe graph-stats -root "%TARGET%"
pidx.exe unresolved -root "%TARGET%" 5000 > unres.txt
echo total unresolved or ambiguous edges (see unres.txt):
find /c /v "" unres.txt
echo == remaining unresolved or ambiguous (first 30)
pidx.exe unresolved -root "%TARGET%" 30
echo == are the missing DOM ids really missing from every html file?
findstr /S /I /N /C:"stat-success" "%TARGET%\Pages\*.html" || echo stat-success: NOT FOUND in any html
findstr /S /I /N /C:"period-year" "%TARGET%\Pages\*.html" || echo period-year: NOT FOUND in any html
echo == where are openEditModal / openDeleteModal defined?
findstr /S /I /N /C:"openEditModal" "%TARGET%\Pages\*.js" "%TARGET%\Pages\*.html"
findstr /S /I /N /C:"openDeleteModal" "%TARGET%\Pages\*.js" "%TARGET%\Pages\*.html"
echo == context: login task (no admin dashboard noise; module-scope section for Login.js)
pidx.exe context -root "%TARGET%" "Modify login to authorize users and redirect by role"
echo == bench: login
pidx.exe bench -root "%TARGET%" "Modify login to authorize users and redirect by role"
echo == context: student application task (should stay as good as before)
pidx.exe context -root "%TARGET%" "Student application form submission should validate input before saving"
echo == bench: student application
pidx.exe bench -root "%TARGET%" "Student application form submission should validate input before saving"
echo == done
