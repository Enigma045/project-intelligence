@echo off
cd /d "%~dp0"
echo == apply phase6 patches (safe to re-run)
go run ./tools/applypatch -root . patches/phase6
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
echo == task context (Go internals like webCtx.add must be gone)
pidx.exe context -root . "Modify the registration form to validate the user's email before submitting it"
echo.
echo ===== PHASE 6: EDIT ENGINE =====
echo == 1. dry run (nothing written)
pidx.exe edit -root . demo/requests/000-dry-run.json
echo == 2. validate-email edit: 1 text replace + replace_symbol + insert_after_symbol across 2 files
pidx.exe edit -root . demo/requests/001-validate-email.json
echo == register.js after edit
type demo\web\register.js
echo == outline (index already updated)
pidx.exe outline -root . demo/web/register.js
echo == callers of isValidEmail
pidx.exe callers -root . isValidEmail
echo == source of the new function
pidx.exe source -root . showEmailError
echo == 3. create a new file
pidx.exe edit -root . demo/requests/002-create-data.json
echo == 4. REJECT: stale base_hash and missing text (all errors reported together)
pidx.exe edit -root . demo/requests/003-reject-bad.json
echo == 5. REJECT: edit would produce invalid JSON
pidx.exe edit -root . demo/requests/004-reject-json.json
echo == 6. validation command fails: automatic rollback
pidx.exe edit -root . demo/requests/005-test-fails.json
echo == register.js must NOT contain the TEMP marker
findstr /C:"TEMP marker" demo\web\register.js
echo (no match above means the rollback worked)
echo == 7. undo: removes demo/data.json
pidx.exe undo -root .
echo == 8. undo: reverts the validate-email edit
pidx.exe undo -root .
echo == files restored
type demo\web\register.js
echo == final scan (expect unchanged or small touch)
pidx.exe scan -root .
echo == done
