@echo off
cd /d "%~dp0"
echo == apply phase5 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase5
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
echo == scan
pidx.exe scan -root .
echo == scan again (expect all unchanged)
pidx.exe scan -root .
echo == graph-stats
pidx.exe graph-stats -root .
echo == callers of submitForm (now includes the button listener)
pidx.exe callers -root . submitForm
echo == web map (class styles now point at elements)
pidx.exe web -root . demo/web/register.html
echo == generate progress.md and module shards
pidx.exe progress -root .
echo == progress.md
type progress.md
echo == task context: registration email validation
pidx.exe context -root . "Modify the registration form to validate the user's email before submitting it"
echo == task context: incremental scan impact (Go)
pidx.exe context -root . "scan detects changed files and computes impact of signature changes" budget=1500
echo == source of submitForm by id
pidx.exe source -root . function:demo/web/register.js#submitForm
echo == source by range
pidx.exe source -root . demo/web/register.js:1-3
echo == progress again (expect rewritten=false)
pidx.exe progress -root .
echo == done
