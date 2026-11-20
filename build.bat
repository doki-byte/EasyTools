@echo off
setlocal enabledelayedexpansion
rem ============================================================================
rem EasyTools build script (Windows)
rem
rem Usage:
rem   build.bat                    Build default target (windows/amd64) + optional UPX
rem   build.bat windows/386        Build 32-bit Windows binary
rem   build.bat windows/arm64      Build Windows ARM64 binary
rem
rem macOS / Linux binaries must be built on the target OS (Wails needs the
rem native WebView): run ./build.sh on macOS/Linux, or push a v* tag to let
rem .github/workflows/build.yml produce all four platform packages.
rem
rem Env:
rem   EASYTOOLS_SKIP_UPX=1  Skip UPX compression (auto-skipped if UPX missing)
rem ============================================================================

set "PLATFORM=%~1"
if "%PLATFORM%"=="" set "PLATFORM=windows/amd64"

echo ==============================================
echo  EasyTools build
echo  target platform: %PLATFORM%
echo ==============================================

rem Non-Windows targets cannot be packaged on a Windows host
if not "%PLATFORM:darwin=%"=="%PLATFORM%" (
    echo [hint] macOS binaries must be built on macOS.
    echo        Copy the project to a Mac and run: ./build.sh mac %2
    echo        Or push a v* tag to trigger GitHub Actions packaging.
    exit /b 0
)
if not "%PLATFORM:linux=%"=="%PLATFORM%" (
    echo [hint] Linux binaries must be built on Linux.
    echo        Run on Linux: ./build.sh linux %2
    echo        Or push a v* tag to trigger GitHub Actions packaging.
    exit /b 0
)

echo [1/4] Checking toolchain...
where go >nul 2>nul || (echo [error] go not found. Install Go and add it to PATH. & exit /b 1)
where npm >nul 2>nul || (echo [error] npm not found. Install Node.js first. & exit /b 1)
where wails >nul 2>nul || (echo [error] wails CLI not found. Run: go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2 & exit /b 1)

echo [2/4] Preparing frontend dependencies...
if not exist "frontend\node_modules" (
    pushd frontend
    call npm install
    if errorlevel 1 (
        echo npm install failed!
        popd
        exit /b 1
    )
    popd
) else (
    echo frontend dependencies already exist, skipping npm install
)

echo [3/4] Building Wails app (%PLATFORM%)...
call wails build --clean --platform %PLATFORM% --trimpath -ldflags="-w -s"
if errorlevel 1 (
    echo Wails build failed!
    exit /b %errorlevel%
)

echo [4/4] Compressing and organizing output...
rem Output subfolder is derived from the platform argument:
rem windows/amd64 -> windows_amd64, windows/386 -> windows_386
set "OUTDIR="
for /f "tokens=1,2 delims=/" %%a in ("%PLATFORM%") do (
    if not "%%b"=="" (set "OUTDIR=%%a_%%b") else (set "OUTDIR=%%a_amd64")
)
if "%OUTDIR%"=="" set "OUTDIR=windows_amd64"

set "UPX_ENABLED=1"
if "%EASYTOOLS_SKIP_UPX%"=="1" set "UPX_ENABLED=0"
where upx >nul 2>nul || set "UPX_ENABLED=0"

if exist "build\bin\EasyTools.exe" (
    if "%UPX_ENABLED%"=="1" (
        echo Compressing with UPX - default level...
        upx "build\bin\EasyTools.exe"
        if errorlevel 1 (
            echo [warn] UPX compression failed - skipped, output is fine
        )
    ) else (
        echo [hint] UPX compression skipped - UPX missing or EASYTOOLS_SKIP_UPX=1
    )
    mkdir "build\bin\%OUTDIR%" >nul 2>nul
    move /y "build\bin\EasyTools.exe" "build\bin\%OUTDIR%\EasyTools.exe" >nul
    set "FINAL=build\bin\%OUTDIR%\EasyTools.exe"
) else (
    echo Executable not found at build\bin\EasyTools.exe
    exit /b 1
)

echo.
echo Build completed!
echo Final executable: %FINAL%
pause
