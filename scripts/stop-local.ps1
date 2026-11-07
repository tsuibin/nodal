# Nodal 本机演示停止脚本（Windows）
#
# 用法：.\scripts\stop-local.ps1
#
# 回收顺序：pid 文件 → 监听端口 → 进程特征兜底（只认 nodal-server.exe 和跑 vite.js 的 node）。

param(
    [int]$BackendPort = 8080,
    [int]$FrontendPort = 5173
)

$Root = Split-Path $PSScriptRoot -Parent
$PidFile = Join-Path $Root "backend\data\local-pids.json"

function Stop-Id([int]$Id, [string]$Why) {
    if ($Id -le 0) { return }
    $p = Get-Process -Id $Id -ErrorAction SilentlyContinue
    if ($p) {
        Stop-Process -Id $Id -Force -ErrorAction SilentlyContinue
        Write-Host "已停止 $($p.ProcessName)（PID $Id，$Why）"
    }
}

# ---- 1) pid 文件 ----
if (Test-Path $PidFile) {
    $info = Get-Content -LiteralPath $PidFile -Raw | ConvertFrom-Json
    Stop-Id ([int]$info.backend_pid) "后端"
    Stop-Id ([int]$info.frontend_pid) "前端"
    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

# ---- 2) 监听端口 ----
foreach ($port in @($BackendPort, $FrontendPort)) {
    $conns = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    foreach ($conn in $conns) { Stop-Id ([int]$conn.OwningProcess) ":$port" }
}

# ---- 3) 进程特征兜底 ----
# nodal-server.exe 只有一个；vite 用「命令行里带 vite.js 且 --port 是本脚本的前端口」认，
# 避免误杀本机其它 node 进程（命令行里的相对路径不含项目名）。
$procs = Get-CimInstance Win32_Process -Filter "Name='nodal-server.exe'" -ErrorAction SilentlyContinue
foreach ($p in $procs) { Stop-Id ([int]$p.ProcessId) "nodal-server.exe" }

$nodes = Get-CimInstance Win32_Process -Filter "Name='node.exe'" -ErrorAction SilentlyContinue |
    Where-Object {
        $_.CommandLine -like "*vite.js*" -and (
            $_.CommandLine -like "*--port $FrontendPort*" -or $_.CommandLine -like "*--port=$FrontendPort*"
        )
    }
foreach ($p in $nodes) { Stop-Id ([int]$p.ProcessId) "vite" }

Start-Sleep -Seconds 2
foreach ($port in @($BackendPort, $FrontendPort)) {
    $still = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    if ($still) { Write-Host "警告：:$port 仍在监听" } else { Write-Host ":$port 已释放" }
}
