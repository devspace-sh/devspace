@echo off
set INSTALL_DIR=%1

if not exist %INSTALL_DIR% mkdir %INSTALL_DIR% || echo "Unable to create install directory. Try again with ADMIN rights." && exit /B 1

reg Query "HKLM\Hardware\Description\System\CentralProcessor\0" | find /i "x86" > NUL && set ARCH_SUFFIX=386 || set ARCH_SUFFIX=amd64

set LATEST_RELEASE_API_URL=https://api.github.com/repos/covexo/devspace/releases/latest
set RELEASE_KEY=tag_name

for /f tokens^=4^ delims^=^<^"^= %%a in ('curl -s %%LATEST_RELEASE_API_URL%% ^| findstr /R /C:%%RELEASE_KEY%%') do (set LATEST_VERSION=%%a)

set DEVSPACE_EXE="https://github.com/covexo/devspace/releases/download/%LATEST_VERSION%/devspace-windows-%ARCH_SUFFIX%.exe"

echo 1. Downloading executable...
set INSTALL_PATH=%INSTALL_DIR%\devspace.exe
if not exist %INSTALL_PATH% curl -L %DEVSPACE_EXE% >INSTALL_PATH || echo Unable to download latest release && exit /B 1

echo 2. Running installation...
start /WAIT /D %INSTALL_DIR% /B devspace.exe "install"

if "%errorlevel%" == "0" (
    echo "Installation successful!"
) else (
    echo "Installation failed!"
    exit /B %errorlevel%
)
