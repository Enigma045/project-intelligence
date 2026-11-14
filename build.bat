@echo off
cd /d "%~dp0"
go mod tidy
if errorlevel 1 exit /b 1
go build -o pidx.exe ./cmd/pidx
if errorlevel 1 exit /b 1
go test ./...
if errorlevel 1 exit /b 1
echo Build OK. Try: pidx.exe scan -root .
