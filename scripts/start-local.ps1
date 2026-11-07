# Nodal 本机演示启动脚本（Windows / SQLite）
#
# 用法：
#   .\scripts\start-local.ps1
#   .\scripts\start-local.ps1 -RuntimeRoot D:\runtime -BackendPort 8080 -FrontendPort 5173
#
# 做的事：
#   1) 用便携 Go 编译后端（已有 nodal-server.exe 就跳过）；
#   2) 以 backend 为工作目录启动后端，SQLite 落在 backend\data\nodal.db，日志在 backend\data\；
#   3) 启动 Vite dev server，/api 反代到后端；
#   4) 把两个进程号写到 backend\data\local-pids.json，交给 stop-local.ps1 回收。
#
# 说明：Windows 上 REPL 代码执行默认关闭（NODAL_REPL_ENABLED=false），
# 因为它的隔离依赖 Linux 的 unshare / 用户命名空间 / tmpfs。

param(
    [string]$RuntimeRoot = (Join-Path (Split-Path $PSScriptRoot -Parent | Split-Path -Parent) "_runtime"),
    [int]$BackendPort = 8080,
    [int]$FrontendPort = 5173,
    [switch]$Rebuild
)

$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent
$GoExe = Join-Path $RuntimeRoot "go\bin\go.exe"
$NodeExe = Join-Path $RuntimeRoot "node-v22.23.4-win-x64\node.exe"
$ServerExe = Join-Path $RuntimeRoot "nodal-server.exe"
$DataDir = Join-Path $Root "backend\data"
$PidFile = Join-Path $DataDir "local-pids.json"

New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

function Test-PortOpen([int]$Port) {
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $client.Connect("127.0.0.1", $Port)
        $client.Close()
        return $true
    } catch {
        return $false
    }
}

# ---- 1) 后端二进制 ----
if ($Rebuild -or -not (Test-Path $ServerExe)) {
    if (-not (Test-Path $GoExe)) { throw "找不到 Go 工具链：$GoExe（用 -RuntimeRoot 指定）" }
    Write-Host "编译后端 …"
    $env:GOPROXY = "https://goproxy.cn,direct"
    $env:GOTOOLCHAIN = "local"
    $env:GOPATH = Join-Path $RuntimeRoot "gopath"
    $env:GOMODCACHE = Join-Path $RuntimeRoot "gopath\pkg\mod"
    # 必须先进入 backend：`./cmd/server` 是相对当前目录的 package 路径
    Push-Location (Join-Path $Root "backend")
    try {
        & $GoExe build -o $ServerExe ./cmd/server 2>&1 | ForEach-Object { "  $_" }
        if ($LASTEXITCODE -ne 0) { throw "后端编译失败" }
    } finally {
        Pop-Location
    }
}

# ---- 2) 后端进程 ----
$backendPid = 0
if (Test-PortOpen $BackendPort) {
    Write-Host "后端已在 :$BackendPort 运行，跳过启动"
} else {
    $env:NODAL_ENV = "dev"
    $env:NODAL_HTTP_ADDR = ":$BackendPort"
    $env:NODAL_DB_DRIVER = "sqlite"
    $env:NODAL_DB_DSN = "data/nodal.db"
    $env:NODAL_STORAGE_DIR = "data/uploads"
    $env:NODAL_EMBED_MODEL = "mock/embed"
    $env:NODAL_REPL_ENABLED = "false"
    $env:NODAL_JWT_SECRET = "nodal-local-demo-secret-please-change-32"
    $env:NODAL_CORS_ORIGINS = "http://localhost:$FrontendPort,http://127.0.0.1:$FrontendPort"

    $proc = Start-Process -FilePath $ServerExe `
        -WorkingDirectory (Join-Path $Root "backend") `
        -RedirectStandardOutput (Join-Path $DataDir "server.out.log") `
        -RedirectStandardError (Join-Path $DataDir "server.err.log") `
        -WindowStyle Hidden -PassThru
    $backendPid = $proc.Id
    Write-Host "后端已启动 :$BackendPort (PID $backendPid)"
}

# ---- 3) 前端进程 ----
$frontendPid = 0
if (Test-PortOpen $FrontendPort) {
    Write-Host "前端已在 :$FrontendPort 运行，跳过启动"
} else {
    if (-not (Test-Path $NodeExe)) { throw "找不到 Node：$NodeExe（用 -RuntimeRoot 指定）" }
    if (-not (Test-Path (Join-Path $Root "frontend\node_modules"))) { throw "前端依赖未安装：先在 frontend 里 npm install" }

    $env:PATH = (Split-Path $NodeExe -Parent) + ";" + $env:PATH
    $env:VITE_DEV_API_TARGET = "http://127.0.0.1:$BackendPort"

    $proc = Start-Process -FilePath $NodeExe `
        -ArgumentList "node_modules\vite\bin\vite.js", "--host", "127.0.0.1", "--port", "$FrontendPort" `
        -WorkingDirectory (Join-Path $Root "frontend") `
        -RedirectStandardOutput (Join-Path $DataDir "frontend.out.log") `
        -RedirectStandardError (Join-Path $DataDir "frontend.err.log") `
        -WindowStyle Hidden -PassThru
    $frontendPid = $proc.Id
    Write-Host "前端已启动 :$FrontendPort (PID $frontendPid)"
}

# ---- 4) 记录进程号并探活 ----
@{
    backend_pid  = $backendPid
    backend_port = $BackendPort
    frontend_pid = $frontendPid
    frontend_port = $FrontendPort
    started_at   = (Get-Date).ToString("s")
} | ConvertTo-Json -Depth 3 | Set-Content -LiteralPath $PidFile -Encoding utf8

Start-Sleep -Seconds 3
try {
    $health = (Invoke-WebRequest -Uri "http://127.0.0.1:$BackendPort/healthz" -UseBasicParsing -TimeoutSec 5).Content
    Write-Host "后端探活：$health"
} catch {
    Write-Host "后端探活失败：$($_.Exception.Message)（看 backend\data\server.err.log）"
}

Write-Host ""
Write-Host "前端：http://127.0.0.1:$FrontendPort"
Write-Host "接口：http://127.0.0.1:$BackendPort/healthz"
Write-Host "日志：backend\data\server.*.log / backend\data\frontend.*.log"
Write-Host "停止：.\scripts\stop-local.ps1"
