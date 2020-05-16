@echo off
setlocal enabledelayedexpansion

WHERE devspace.exe >nul 2>nul
IF %ERRORLEVEL% EQU 0 devspace.exe %* && exit /b %errorlevel%

echo Finishing installation of DevSpace CLI

SET bindir=%~dp0
SET basedir=%~dp0\..
SET indexFile=\index.js

echo "!basedir!\!indexFile!"

IF NOT EXIST "!basedir!\!indexFile!" (
  SET basedir=%~dp0\..\lib\node_modules\devspace

  IF NOT EXIST "!basedir!\!indexFile!" (
    SET basedir=%~dp0\node_modules\devspace

    IF NOT EXIST "!basedir!\!indexFile!" (
      FOR /F "tokens=* USEBACKQ" %%F IN (`npm root -g`) DO (
        SET basedir=%%F\devspace
      )

      IF NOT EXIST "!basedir!\!indexFile!" (
        FOR /F "tokens=* USEBACKQ" %%F IN (`yarn global dir`) DO (
          SET basedir=%%F\node_modules\devspace
        )

        IF NOT EXIST "!basedir!\!indexFile!" (
          echo Unable to find global npm/yarn dir
          exit /b 1
        )
      )
    )
  )
)

echo Running: node "!basedir!\!indexFile!" finish-install "!bindir!\"
node "!basedir!\!indexFile!" finish-install "!bindir!\" && (!bindir!\devspace.exe %* 2> nul || !bindir!\..\..\.bin\devspace.exe %* 2> nul || devspace.exe %*)
