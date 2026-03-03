#!/usr/bin/env pwsh
# Makefile.ps1 - аналог Makefile для PowerShell

param(
    [Parameter(Position=0)]
    [string]$Target = "help"
)

$BINARY_NAME = "projectT"
$CMD_PATH = ".\cmd\main.go"
$VERSION = if ($env:VERSION) { $env:VERSION } else { "dev" }

function Invoke-Build {
    Write-Host "Building $BINARY_NAME..." -ForegroundColor Cyan
    go build -v -ldflags "-X main.Version=$VERSION" -o "$BINARY_NAME.exe" $CMD_PATH
    Write-Host "Build complete: $BINARY_NAME.exe" -ForegroundColor Green
}

function Invoke-Run {
    Write-Host "Running $BINARY_NAME..." -ForegroundColor Cyan
    go run $CMD_PATH
}

function Invoke-Test {
    Write-Host "Running tests..." -ForegroundColor Cyan
    go test -v -race -cover ./...
}

function Invoke-Clean {
    Write-Host "Cleaning..." -ForegroundColor Cyan
    Remove-Item -Force -ErrorAction SilentlyContinue "$BINARY_NAME.exe"
    Remove-Item -Force -ErrorAction SilentlyContinue coverage.out
    Remove-Item -Force -ErrorAction SilentlyContinue coverage.html
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue dist/
    Write-Host "Clean complete" -ForegroundColor Green
}

function Invoke-Lint {
    Write-Host "Running linter..." -ForegroundColor Cyan
    golangci-lint run ./...
}

function Invoke-PreCommitInstall {
    Write-Host "Installing pre-commit hooks..." -ForegroundColor Cyan
    pre-commit install
    Write-Host "Pre-commit hooks installed" -ForegroundColor Green
}

function Invoke-PreCommitRun {
    Write-Host "Running pre-commit checks..." -ForegroundColor Cyan
    pre-commit run --all-files
}

function Invoke-PreCommitUninstall {
    Write-Host "Uninstalling pre-commit hooks..." -ForegroundColor Cyan
    pre-commit uninstall 2>$null
    Write-Host "Pre-commit hooks uninstalled" -ForegroundColor Green
}

function Invoke-Help {
    Write-Host @"
Available targets:

  build              - Собрать приложение для Windows
  run                - Запустить приложение
  test               - Запустить тесты
  clean              - Очистить артефакты сборки
  lint               - Запустить линтер
  pre-commit-install - Установить pre-commit хуки
  pre-commit-run     - Запустить pre-commit проверки вручную
  pre-commit-uninstall - Удалить pre-commit хуки
  help               - Показать эту справку
"@ -ForegroundColor Yellow
}

switch ($Target) {
    "build" { Invoke-Build }
    "run" { Invoke-Run }
    "test" { Invoke-Test }
    "clean" { Invoke-Clean }
    "lint" { Invoke-Lint }
    "pre-commit-install" { Invoke-PreCommitInstall }
    "pre-commit-run" { Invoke-PreCommitRun }
    "pre-commit-uninstall" { Invoke-PreCommitUninstall }
    "help" { Invoke-Help }
    default {
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Invoke-Help
        exit 1
    }
}
