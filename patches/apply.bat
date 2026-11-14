@echo off
cd /d "%~dp0.."
echo == applying patches
go run ./tools/applypatch -root . patches/phase2
if errorlevel 1 exit /b 1
echo == build
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
echo == test
go test ./...
if errorlevel 1 exit /b 1
echo == scan 1 (builds the graph)
pidx.exe scan -root .
echo == scan 2 (expect zero added/changed)
pidx.exe scan -root .
echo == stats
pidx.exe stats -root .
echo == graph-stats
pidx.exe graph-stats -root .
echo == callers of HashBytes
pidx.exe callers -root . HashBytes
echo == callees of process
pidx.exe callees -root . process
echo == deps of cmd/pidx/main.go
pidx.exe deps -root . cmd/pidx/main.go
echo == first 25 unresolved or ambiguous
pidx.exe unresolved -root . 25
echo Phase 2 done.
