@echo off
echo Starting build process...

echo 1. Cleaning frontend...
cd /d frontend
if exist node_modules rd /s /q node_modules
if exist dist rd /s /q dist

echo 2. Installing frontend dependencies...
call npm install
if %ERRORLEVEL% neq 0 (
    echo npm install failed!
    pause
    exit /b %ERRORLEVEL%
)

echo 3. Returning to root directory...
cd /d ..

echo 4. Building Wails application...
call wails build --trimpath -ldflags="-w -s"
if %ERRORLEVEL% neq 0 (
    echo Wails build failed!
    pause
    exit /b %ERRORLEVEL%
)

echo 5. Compressing executable with UPX...
if exist "build\bin\EasyTools.exe" (
    upx "build\bin\EasyTools.exe"
    if %ERRORLEVEL% neq 0 (
        echo UPX compression failed. Is UPX installed and in your PATH?
        pause
        exit /b %ERRORLEVEL%
    )
) else (
    echo Executable not found at build\bin\EasyTools.exe
    pause
    exit /b 1
)

echo.
echo Build and compression completed successfully!
echo Final executable: build\bin\EasyTools.exe
pause