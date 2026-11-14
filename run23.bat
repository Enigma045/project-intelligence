@echo off
cd /d "%~dp0"
set GIT_TERMINAL_PROMPT=0
set REPO=https://github.com/Enigma045/project-intelligence.git
git --version >nul 2>&1
if errorlevel 1 goto nogit
if exist .git goto haverepo
echo [%time%] git init
git init
git branch -M main
:haverepo
git config user.name >nul 2>&1
if errorlevel 1 goto noident
git config user.email >nul 2>&1
if errorlevel 1 goto noident
git remote get-url origin >nul 2>&1
if errorlevel 1 goto addremote
git remote set-url origin %REPO%
goto afterremote
:addremote
git remote add origin %REPO%
:afterremote
echo [%time%] staging files
git add -A
echo [%time%] files staged for commit:
git status --short
echo [%time%] number of staged files:
git diff --cached --name-only | find /c /v ""
git diff --cached --name-only | findstr /I /R "\.exe$ \.db$ ^Ubaps_Go-main/ ^\.env" >nul
if not errorlevel 1 goto unsafe
git diff --cached --quiet
if errorlevel 1 goto docommit
echo [%time%] nothing new to commit
goto push
:docommit
echo [%time%] commit
git commit -m "Initial commit: pidx project intelligence tool (indexer, resolver, context builder, edit engine, eval harness)"
if errorlevel 1 goto commitfail
:push
git branch -M main
echo [%time%] push to %REPO%
git push -u origin main
if errorlevel 1 goto pushfail
echo [%time%] pushed OK
git log --oneline -n 3
exit /b 0
:nogit
echo ERROR: git is not installed or not on PATH
exit /b 1
:noident
echo ERROR: git has no identity configured. Run these once, with your own details:
echo   git config --global user.name "Your Name"
echo   git config --global user.email "you@example.com"
exit /b 1
:unsafe
echo ERROR: staged files include a build output, database, Ubaps_Go-main file or .env. Nothing was committed.
git diff --cached --name-only | findstr /I /R "\.exe$ \.db$ ^Ubaps_Go-main/ ^\.env"
git reset >nul 2>&1
exit /b 1
:commitfail
echo ERROR: commit failed
exit /b 1
:pushfail
echo ERROR: push failed. Likely causes:
echo   1. the GitHub repo already has commits (for example a README): nothing was overwritten. Tell me whether to rebase onto it
echo   2. no stored GitHub credentials: run  git push -u origin main  once in a normal terminal, then re-run this file
echo   3. the repository does not exist or is not accessible to your account
exit /b 1
