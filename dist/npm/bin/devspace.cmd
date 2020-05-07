@echo off

WHERE devspace.exe >nul 2>nul
IF %ERRORLEVEL% EQU 0 devspace %* && exit /b %errorlevel%

echo Finishing installation of DevSpace CLI

SET basedir=%~dp0\..
SET indexFile=\index.js

echo "%basedir%\%indexFile%"

IF NOT EXIST "%basedir%\%indexFile%" (
  SET basedir=%~dp0\..\lib\node_modules\devspace

  IF NOT EXIST "%basedir%\%indexFile%" (
    FOR /F "tokens=* USEBACKQ" %%F IN (`npm root -g`) DO (
      SET basedir=%%F\devspace
    )

    IF NOT EXIST "%basedir%\%indexFile%" (
      FOR /F "tokens=* USEBACKQ" %%F IN (`yarn global dir`) DO (
        SET basedir=%%F\node_modules\devspace
        IF NOT EXIST "%basedir%\%indexFile%" (
          echo "Unable to find global npm/yarn dir"
          exit /b 1
        )
      )
    )
  )
)

echo Running: node %basedir%\%indexFile% finish-install
node "%basedir%\%indexFile%" finish-install && devspace.exe %* && exit /b %errorlevel%
