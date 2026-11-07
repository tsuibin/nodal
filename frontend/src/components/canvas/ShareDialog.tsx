/**
 * 分享弹窗：创建 read / fork 两种权限链接，复制到剪贴板。
 * 链接只在本画布所有者/可分享身份下创建；模块 9 只负责发起，
 * 真正的只读分享页留给模块 10。
 */
import { useState, type ReactNode } from 'react'
import { useShareLinks } from '../../hooks/useShareLinks'
import type { SharePermission } from '../../types/square'
import { CheckIcon, CloseIcon, ForkIcon, ShareIcon } from '../icons'

export interface ShareDialogProps {
  canvasId: number
  onClose: () => void
}

export function ShareDialog({ canvasId, onClose }: ShareDialogProps) {
  const share = useShareLinks(canvasId)
  const [permission, setPermission] = useState<SharePermission>('read')
  const [expires, setExpires] = useState<number>(0)
  const [copied, setCopied] = useState<string>('')
  const [busy, setBusy] = useState(false)

  const create = async () => {
    setBusy(true)
    try {
      const link = await share.create(permission, expires || undefined)
      if (link) {
        const url = shareURL(link.share_path, link.token)
        setCopied(url)
        await copyText(url)
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="nodal-dialog-backdrop" role="presentation" onMouseDown={onClose}>
      <div
        className="nodal-dialog max-w-2xl"
        role="dialog"
        aria-modal="true"
        aria-label="分享画布"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="nodal-dialog-head">
          <div>
            <p className="nodal-kicker">SHARE</p>
            <h2 className="nodal-dialog-title">把这张画布分享出去</h2>
          </div>
          <button type="button" className="nodal-icon-btn" onClick={onClose} aria-label="关闭">
            <CloseIcon />
          </button>
        </header>

        <div className="space-y-4 p-5">
          <div className="grid gap-2 sm:grid-cols-2">
            <PermissionCard
              active={permission === 'read'}
              icon={<ShareIcon />}
              title="只读分享"
              description="对方可以查看完整画布，但不能编辑、不能推演。"
              onClick={() => setPermission('read')}
            />
            <PermissionCard
              active={permission === 'fork'}
              icon={<ForkIcon />}
              title="可 Fork"
              description="对方可以一键复制成自己的画布，继续修改。"
              onClick={() => setPermission('fork')}
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-[var(--nodal-muted)]">有效期</span>
            {[
              { value: 0, label: '永久' },
              { value: 24, label: '24 小时' },
              { value: 72, label: '3 天' },
              { value: 720, label: '30 天' },
            ].map((item) => (
              <button
                key={item.value}
                type="button"
                className={expires === item.value ? 'nodal-chip nodal-chip-active' : 'nodal-chip'}
                onClick={() => setExpires(item.value)}
              >
                {item.label}
              </button>
            ))}
            <button
              type="button"
              className="nodal-btn nodal-btn-primary ml-auto"
              onClick={() => void create()}
              disabled={busy || share.loading}
            >
              创建并复制链接
            </button>
          </div>

          {copied ? (
            <div className="nodal-copy-box">
              <span className="truncate font-mono text-[11px]">{copied}</span>
              <button
                type="button"
                className="nodal-btn nodal-btn-ghost shrink-0"
                onClick={() => void copyText(copied)}
              >
                <CheckIcon width={14} height={14} />
                复制
              </button>
            </div>
          ) : null}

          {share.error ? (
            <p className="text-xs text-counter">{share.error.message}</p>
          ) : null}

          <div className="space-y-2">
            <p className="text-[11px] font-semibold tracking-wide text-[var(--nodal-muted)]">
              已有链接（{share.links.length}）
            </p>
            {share.links.length === 0 ? (
              <p className="text-xs text-[var(--nodal-muted)]">还没有分享链接。</p>
            ) : (
              share.links.map((link) => {
                const url = shareURL(link.share_path, link.token)
                return (
                  <div key={link.id} className="nodal-link-row">
                    <span className="rounded-full border border-[var(--nodal-border)] px-2 py-0.5 text-[10px]">
                      {link.permission === 'read' ? '只读' : '可 Fork'}
                    </span>
                    <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-[var(--nodal-muted)]">
                      {url}
                    </span>
                    <span className="text-[10px] text-[var(--nodal-muted)]">
                      {link.enabled && !link.expired ? '有效' : '已失效'}
                    </span>
                    <button
                      type="button"
                      className="nodal-btn nodal-btn-ghost"
                      disabled={!link.enabled || link.expired}
                      onClick={() => void copyText(url)}
                    >
                      复制
                    </button>
                    {link.enabled ? (
                      <button
                        type="button"
                        className="nodal-btn nodal-btn-danger"
                        onClick={() => void share.revoke(link.id)}
                      >
                        停用
                      </button>
                    ) : null}
                  </div>
                )
              })
            )}
          </div>
        </div>

        <footer className="nodal-dialog-foot">
          <span className="text-[11px] text-[var(--nodal-muted)]">
            分享链接是只读/可 Fork 权限；画布公开到广场请到资源库操作。
          </span>
          <button type="button" className="nodal-btn nodal-btn-primary ml-auto" onClick={onClose}>
            完成
          </button>
        </footer>
      </div>
    </div>
  )
}

function PermissionCard({
  active,
  icon,
  title,
  description,
  onClick,
}: {
  active: boolean
  icon: ReactNode
  title: string
  description: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-xl border p-3 text-left transition ${
        active
          ? 'border-brand bg-brand/8 shadow-[0_6px_20px_rgb(64_96_255/0.12)]'
          : 'border-[var(--nodal-border)] hover:border-brand/40'
      }`}
    >
      <span className="flex items-center gap-2 text-sm font-semibold">
        {icon}
        {title}
      </span>
      <span className="mt-1 block text-[11px] leading-5 text-[var(--nodal-muted)]">{description}</span>
    </button>
  )
}

function shareURL(path: string, token: string): string {
  const suffix = path || `/share/${token}`
  return `${window.location.origin}${suffix}`
}

async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    return
  } catch {
    /* 无剪贴板权限时走隐藏 textarea + execCommand，不弹阻塞式 prompt */
  }
  try {
    const area = document.createElement('textarea')
    area.value = text
    area.setAttribute('readonly', '')
    area.style.position = 'fixed'
    area.style.opacity = '0'
    document.body.appendChild(area)
    area.select()
    document.execCommand('copy')
    document.body.removeChild(area)
  } catch {
    /* 复制失败也不阻塞操作，用户可手动选中链接 */
  }
}
