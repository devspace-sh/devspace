@echo off

WHERE devspace >nul 2>nul
IF %ERRORLEVEL% EQU 0 devspace %* && exit /b %errorlevel%

echo Finishing installation for DevSpace CLI

for %%F in (%0) do set dirname=%%~dpF

node %dirname%\..\index.js force-install && devspace %*
exit /b %errorlevel%
