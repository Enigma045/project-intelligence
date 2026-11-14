@echo off
cd /d "%~dp0"
echo == apply phase8 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase8
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
echo == scan (version bump: graph rebuild)
pidx.exe scan -root .
echo == scan again (expect all unchanged)
pidx.exe scan -root .
echo == unresolved or ambiguous (expect 0 edges now)
pidx.exe unresolved -root . 25
echo == task context: users table task (should now include table users and no Store internals)
pidx.exe context -root . "Registration should store the email in the users table and validate it"
echo == token benchmark
pidx.exe bench -root . "Registration should store the email in the users table and validate it"
echo == task context: email validation task
pidx.exe context -root . "Modify the registration form to validate the user's email before submitting it"
echo == regenerate progress.md
pidx.exe progress -root .
echo == done
