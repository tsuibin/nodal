/** 卡片列表骨架屏，避免首屏空白和布局跳动。 */
export function LoadingGrid({ count = 6 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: count }).map((_, index) => (
        <div key={index} className="nodal-card h-[248px] animate-pulse overflow-hidden">
          <div className="h-28 border-b border-[var(--nodal-border)] bg-[var(--nodal-surface-soft)]" />
          <div className="space-y-3 p-4">
            <div className="h-4 w-2/3 rounded bg-[var(--nodal-surface-soft)]" />
            <div className="h-3 w-1/2 rounded bg-[var(--nodal-surface-soft)]" />
            <div className="mt-6 h-3 w-1/3 rounded bg-[var(--nodal-surface-soft)]" />
          </div>
        </div>
      ))}
    </div>
  )
}
