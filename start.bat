@echo off
setlocal
echo ==========================================
echo   Levantando Aplicacion Websokk
echo ==========================================

echo [1/3] Construyendo y arrancando contenedores...
docker-compose up --build -d

echo [2/3] Esperando a que los servicios esten listos...
timeout /t 5 /nobreak > nul

echo [3/3] Abriendo la aplicacion en el navegador...
start http://localhost:8080

echo.
echo Aplicacion iniciada correctamente!
echo Para ver los logs, usa: docker-compose logs -f
echo Para detener, usa: stop.bat
echo.
pause
