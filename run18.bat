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
go run ./tools/applypatch -root . patches/phase16
if errorlevel 1 exit /b 1
echo [%time%] build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo [%time%] test
go test ./...
if errorlevel 1 exit /b 1
echo [%time%] scan target (extractor version bump: full re-extract)
pidx.exe scan -root "%TARGET%"
echo [%time%] graph-stats (before: calls/dynamic 322, calls/resolved 249)
pidx.exe graph-stats -root "%TARGET%"
echo [%time%] callers of Create_user_log (should list many call sites now)
pidx.exe callers -root "%TARGET%" Create_user_log
echo [%time%] edge evaluation (the Create_user_log miss should be fixed)
pidx.exe eval -root "%TARGET%" edges labels=eval\ubaps.edges.json
echo [%time%] gate (baseline NOT updated)
pidx.exe eval -root "%TARGET%" all labels=eval\ubaps.edges.json tasks=eval\ubaps.tasks.json baseline=eval\ubaps.baseline.json
echo [%time%] answers to the skipped questions
echo -- package clause of Audit_logs:
findstr /N /B /C:"package" "%TARGET%\Audit_logs\*.go"
echo -- how Finance.go imports it:
findstr /N /C:"Audit_logs" "%TARGET%\Routes\Finance.go"
echo -- API_BASE_URL definition:
findstr /S /N /C:"API_BASE_URL =" /C:"API_BASE_URL=" "%TARGET%\Pages\*.js" "%TARGET%\Pages\*.html" || echo (no definition found)
echo -- rs-toggle-btn anywhere:
findstr /S /N /C:"rs-toggle-btn" "%TARGET%\Pages\*.js" "%TARGET%\Pages\*.html" || echo (not found)
echo -- openModalRequest anywhere:
findstr /S /N /C:"openModalRequest" "%TARGET%\Pages\*.js" "%TARGET%\Pages\*.html" || echo (not found)
echo [%time%] sampling 30 new unlinked edges that could really link (no chain calls, no value edges)
pidx.exe eval -root "%TARGET%" sample out=eval\ubaps.edges.json n=30 mode=unlinked seed=2 types=calls,requests_api,listens,handles_route,queries_table
pidx.exe eval -root "%TARGET%" sheet labels=eval\ubaps.edges.json out=eval\review.txt
echo [%time%] done
