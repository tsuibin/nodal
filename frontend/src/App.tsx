import { useEffect, useState } from 'react';
import { fetchHealth } from '@/lib/api';

/**
 * 模块 0 占位首页
 * 目标：验证脚手架可用 —— Vite + React + TS + Tailwind v4 + 设计令牌 + 后端联通
 * 真正的首页 / 资源库 / 广场在模块 8 实现
 */
type HealthState = 'checking' | 'ok' | 'down';

function NodeDot({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex items-center gap-2 text-xs text-nodal-muted">
      <span
        className="inline-block h-2 w-2 rounded-full"
        style={{ backgroundColor: color }}
      />
      {label}
    </div>
  );
}

export default function App() {
  const [health, setHealth] = useState<HealthState>('checking');

  useEffect(() => {
    let alive = true;
    fetchHealth()
      .then(() => alive && setHealth('ok'))
      .catch(() => alive && setHealth('down'));
    return () => {
      alive = false;
    };
  }, []);

  const healthText =
    health === 'checking' ? '连接后端…' : health === 'ok' ? '后端已连接' : '后端未启动';

  return (
    <main className="min-h-full bg-nodal-bg text-nodal-text flex items-center justify-center px-8">
      <div className="max-w-[560px] w-full">
        <h1 className="text-6xl font-semibold tracking-tight text-nodal-primary">Nodal</h1>
        <p className="mt-4 text-2xl">思维不再是一条直线</p>

        <div className="mt-8 h-px w-16 bg-nodal-primary" />

        <p className="mt-6 text-sm leading-7 text-nodal-muted">
          无限画布式 AI 思维推演引擎。
          <br />
          搭建假设 · 分叉思路 · 多智能体辩论
        </p>

        <div className="mt-10 flex gap-6">
          <NodeDot color="#4060ff" label="推导" />
          <NodeDot color="#ff6b6b" label="反例" />
          <NodeDot color="#36c9a5" label="佐证" />
          <NodeDot color="#ffc857" label="并行" />
        </div>

        <div className="mt-10 flex items-center gap-2 text-[11px] tracking-wider text-nodal-muted font-mono uppercase">
          <span
            className="inline-block h-1.5 w-1.5 rounded-full"
            style={{
              backgroundColor:
                health === 'ok' ? '#36c9a5' : health === 'down' ? '#ff6b6b' : '#9ca3af',
            }}
          />
          {healthText}
          <span className="text-nodal-border">|</span>
          <span>MODULE 0 · SCAFFOLD</span>
        </div>
      </div>
    </main>
  );
}
