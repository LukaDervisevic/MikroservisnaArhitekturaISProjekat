import { NavLink, Outlet } from 'react-router-dom'

const navigation = [
  { to: '/', label: 'Overview', end: true },
  { to: '/lecturers', label: 'Lecturers' },
  { to: '/locations', label: 'Locations' },
  { to: '/events', label: 'Events' },
  { to: '/lectures', label: 'Lectures' },
  { to: '/sourced-events', label: 'Event Sourcing' },
]

export function Layout() {
  return (
    <div className="flex min-h-full flex-col lg:flex-row">
      <aside className="shrink-0 bg-slate-900 lg:w-60">
        <div className="px-5 py-5">
          <p className="text-sm font-semibold tracking-tight text-white">
            MAIS Admin
          </p>
          <p className="mt-0.5 text-xs text-slate-400">
            Conference management
          </p>
        </div>
        <nav className="flex gap-1 overflow-x-auto px-3 pb-4 lg:flex-col lg:overflow-visible">
          {navigation.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `whitespace-nowrap rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-slate-800 text-white'
                    : 'text-slate-300 hover:bg-slate-800/60 hover:text-white'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <main className="min-w-0 flex-1 px-5 py-6 lg:px-8 lg:py-8">
        <div className="mx-auto max-w-6xl">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
