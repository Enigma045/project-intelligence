@echo off
cd /d "%~dp0"
echo == apply phase4 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase4
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
echo == scan (extractor version changed: forced re-extract + graph rebuild)
pidx.exe scan -root .
echo == scan again (expect all unchanged)
pidx.exe scan -root .
echo == stats
pidx.exe stats -root .
echo == graph-stats
pidx.exe graph-stats -root .
echo == web map of demo/web/register.html
pidx.exe web -root . demo/web/register.html
echo == who references submitBtn
pidx.exe dom-refs -root . submitBtn
echo == who references emailError
pidx.exe dom-refs -root . emailError
echo == callers of submitForm
pidx.exe callers -root . submitForm
echo == unresolved or ambiguous
pidx.exe unresolved -root . 20
echo == done
