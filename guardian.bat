@echo off
rem =============================================================================
rem Guardian Autonomous Enterprise SRE Platform (Windows Launcher)
rem =============================================================================

setlocal enabledelayedexpansion
set SCRIPT_DIR=%~dp0
set BIN_ROOT=%SCRIPT_DIR%guardian.exe
set BIN_SUB=%SCRIPT_DIR%bin\guardian.exe

if exist "%BIN_ROOT%" (
    set GUARDIAN_EXE=%BIN_ROOT%
) else if exist "%BIN_SUB%" (
    set GUARDIAN_EXE=%BIN_SUB%
) else (
    echo [ERRO] Executavel guardian.exe nao encontrado no diretorio raiz ou em bin\
    echo Compile utilizando 'make exe' ou 'go build -o guardian.exe ./cmd/guardian'
    pause
    exit /b 1
)

if "%~1"=="" (
    echo [Guardian] Iniciando Guardian Desktop no Windows...
    start "" "%GUARDIAN_EXE%" -gui
) else (
    "%GUARDIAN_EXE%" %*
)
