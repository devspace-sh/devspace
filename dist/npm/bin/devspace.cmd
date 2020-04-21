@echo off

WHERE devspace.exe >nul 2>nul
IF %ERRORLEVEL% EQU 0 devspace %* && exit /b %errorlevel%

echo Finishing installation of DevSpace CLI

FOR /F "tokens=* USEBACKQ" %%F IN (`npm root -g`) DO (
  SET basedir=%%F
)

SET indexFile=devspace\index.js

IF NOT EXIST "%basedir%\%indexFile%" (
  FOR /F "tokens=* USEBACKQ" %%F IN (`npm root -g`) DO (
    SET basedir=%%F\node_modules
    IF NOT EXIST "%basedir%\%indexFile%" (
      echo "Unable to find global npm/yarn dir"
      exit /b 1
    )
  )
)

echo Running: node %basedir%\%indexFile% finish-install
node %basedir%\%indexFile% finish-install && devspace.exe %* && exit /b %errorlevel%
