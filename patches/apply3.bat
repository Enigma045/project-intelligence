@echo off
cd /d "%~dp0.."
echo == applying phase3 patches
go run ./tools/applypatch -root . patches/phase3
if errorlevel 1 exit /b 1
echo == build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo == test
go test ./...
if errorlevel 1 exit /b 1
echo == scan 1 (extractor version changed: forced re-extract + graph rebuild)
pidx.exe scan -root .
echo == scan 2 (expect all unchanged)
pidx.exe scan -root .
echo == graph-stats (compare: unresolved was 128)
pidx.exe graph-stats -root .
echo == remaining unresolved/ambiguous
pidx.exe unresolved -root . 25
echo == sample of dynamic
pidx.exe unresolved -root . 12 dynamic
echo == callers of foo before signature change
pidx.exe callers -root . foo
echo == apply demo patch: foo(x) becomes foo(x, y)
go run ./tools/applypatch -root . patches/demo
if errorlevel 1 exit /b 1
echo == scan 3 (expect IMPACT listing demo/b.py)
pidx.exe scan -root .
echo Phase 3 done.
