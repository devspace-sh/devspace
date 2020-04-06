echo "Finishing installation for DevSpace CLI"

node ..\index.js force-install && devspace %*
exit /b %errorlevel%
