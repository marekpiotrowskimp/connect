@echo off
REM Connect V3 start script for Windows
REM Starts connectd service and launches the desktop UI.

setlocal

set "SCRIPT_DIR=%~dp0"
set "BIN_DIR=%SCRIPT_DIR%bin"
set "CONNECTD=%BIN_DIR%\connectd-windows-amd64.exe"
set "CONNECT_UI=%BIN_DIR%\connect-ui.exe"
set "LOG=%TEMP%\connectd.log"

REM Check for binaries in current directory first
if exist "%~dp0connectd-windows-amd64.exe" set "CONNECTD=%~dp0connectd-windows-amd64.exe"
if exist "%~dp0connect-ui.exe" set "CONNECT_UI=%~dp0connect-ui.exe"

echo Connect V3 -- starting service...

REM Allow connectd through Windows Firewall (requires admin; silently ignored otherwise)
netsh advfirewall firewall add rule name="Connect Service (connectd)" dir=in action=allow program="%CONNECTD%" enable=yes >NUL 2>NUL

REM Always restart connectd to ensure latest binary is used
taskkill /F /IM connectd-windows-amd64.exe 2>NUL
timeout /T 1 /NOBREAK >NUL

if not exist "%CONNECTD%" (
    echo connectd binary not found at %CONNECTD%
    echo Run 'make build-connectd' first.
    pause
    exit /b 1
)
echo Starting connectd...
start "" /B "%CONNECTD%" > "%LOG%" 2>&1
timeout /T 2 /NOBREAK >NUL

if not exist "%CONNECT_UI%" (
    echo connect-ui binary not found at %CONNECT_UI%
    echo Run 'make build-ui-windows' first.
    pause
    exit /b 1
)
echo Launching connect-ui...
start "" "%CONNECT_UI%"

endlocal
