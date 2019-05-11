@echo off

:: Configure to use minikube docker
2>NUL @FOR /f "tokens=* delims=" %%i IN ('minikube docker-env') DO %%i

:: Build the docker image
docker build -t %1 . -f custom/Dockerfile
