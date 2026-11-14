@echo off
cd /d "%~dp0"
set TARGET=Ubaps_Go-main
if exist "%TARGET%\" goto found
set TARGET=..\Ubaps_Go-main
if exist "%TARGET%\" goto found
echo ERROR: Ubaps_Go-main not found
exit /b 1
:found
echo == apply phase11 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase11
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
echo == scan target (version bump rebuilds the graph)
pidx.exe scan -root "%TARGET%"
echo == scan again (expect all unchanged)
pidx.exe scan -root "%TARGET%"
echo == graph-stats (compare: calls/inferred 98, calls/value 969, calls/dynamic 273)
pidx.exe graph-stats -root "%TARGET%"
pidx.exe unresolved -root "%TARGET%" 5000 > unres.txt
echo total unresolved or ambiguous edges (was 148):
find /c /v "" unres.txt
echo == who still resolves to AutoTime.Scan (expect nothing)
pidx.exe callers -root "%TARGET%" Scan
echo == why is Deans_Decisions.js not linked to Deans_Decisions.html? script tags of that page:
findstr /I /N /C:"<script" "%TARGET%\Pages\Html\student\Protected\Deans_Decisions.html"
echo == web map head for that page
pidx.exe web -root "%TARGET%" Pages/Html/student/Protected/Deans_Decisions.html > deans_web.txt
findstr /C:"loads_script" /C:"loads_style" deans_web.txt
echo == context: login task (expect no Get_*_Logs or report functions)
pidx.exe context -root "%TARGET%" "Modify login to authorize users and redirect by role"
echo == bench: login
pidx.exe bench -root "%TARGET%" "Modify login to authorize users and redirect by role"
echo == context: student application task (SubmitForm handler and UpdateApplication must be back)
pidx.exe context -root "%TARGET%" "Student application form submission should validate input before saving"
echo == bench: student application
pidx.exe bench -root "%TARGET%" "Student application form submission should validate input before saving"
echo == done
