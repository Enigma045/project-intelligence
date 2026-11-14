@echo off
cd /d "%~dp0"
echo == apply phase7 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase7
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
echo == scan (extractor version bump: re-extract and rebuild graph)
pidx.exe scan -root .
echo == scan again (expect all unchanged)
pidx.exe scan -root .
echo == graph-stats
pidx.exe graph-stats -root .
echo == routes: handler, queries and JS callers
pidx.exe routes -root .
echo == db: tables, columns, foreign keys and who queries them
pidx.exe db -root . users
pidx.exe db -root . sessions
echo == db: the project's own embedded schema
pidx.exe db -root . symbols
echo == end-to-end task context: UI -^> API -^> handler -^> table
pidx.exe context -root . "Registration should store the email in the users table and validate it"
echo == token benchmark
pidx.exe bench -root . "Registration should store the email in the users table and validate it"
echo == regenerate progress.md
pidx.exe progress -root .
echo == unresolved or ambiguous (review these)
pidx.exe unresolved -root . 25
echo == done
