@echo off

WHERE devspace >nul 2>nul
IF %ERRORLEVEL% EQU 0 devspace %* && exit /b %errorlevel%

echo Finishing installation for DevSpace CLI

FOR /F "tokens=* USEBACKQ" %%F IN (`npm root -g`) DO (
SET basedir=%%F
)

node %basedir%\devspace\index.js force-install && devspace.exe %*
