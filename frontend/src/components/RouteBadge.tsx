/**
 * Shows which gateway route produced the data on screen. This is a teaching
 * tool as much as an admin panel — several resources are reachable through
 * more than one route (a filterable `list` and a narrower dedicated one), and
 * making the active one visible keeps that mapping obvious.
 */
export function RouteBadge({ path }: { path: string }) {
  return (
    <code className="rounded bg-slate-100 px-2 py-1 font-mono text-xs text-slate-600">
      POST {path}
    </code>
  )
}
