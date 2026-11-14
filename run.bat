@echo off
cd /d "%~dp0"
echo == go mod tidy
go mod tidy
if errorlevel 1 exit /b 1
echo == build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo == test
go test ./...
if errorlevel 1 exit /b 1
echo == reset demo files
go run ./tools/resetdemo
if errorlevel 1 exit /b 1
echo == scan
pidx.exe scan -root .
echo == scan again (expect all unchanged)
pidx.exe scan -root .
echo == stats
pidx.exe stats -root .
echo == graph-stats (Phase 2 had calls/unresolved 128)
pidx.exe graph-stats -root .
echo == remaining unresolved or ambiguous
pidx.exe unresolved -root . 25
echo == sample of dynamic edges
pidx.exe unresolved -root . 12 dynamic
echo == sample of value edges
pidx.exe unresolved -root . 12 value
echo == callers of foo before signature change
pidx.exe callers -root . foo
echo == change foo(x) to foo(x, y)
go run ./tools/applypatch -root . patches/demo
if errorlevel 1 exit /b 1
echo == scan (expect IMPACT listing demo/b.py)
pidx.exe scan -root .
echo == done
