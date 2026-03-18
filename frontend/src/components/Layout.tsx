import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import {
  LayoutDashboard, Activity, Bell, Users, LogOut, Radio, Globe, Plug,
} from 'lucide-react'
import clsx from 'clsx'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard', end: true },
  { to: '/monitors', icon: Activity, label: 'Monitors' },
  { to: '/notifications', icon: Bell, label: 'Notifications' },
]

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div className="flex h-screen overflow-hidden bg-gray-950">
      {/* Sidebar */}
      <aside className="w-60 flex-shrink-0 flex flex-col bg-gray-900 border-r border-gray-800">
        {/* Logo */}
        <div className="flex items-center gap-2.5 px-5 py-5 border-b border-gray-800">
          <div className="w-8 h-8 bg-indigo-600 rounded-lg flex items-center justify-center">
            <Radio className="w-4 h-4 text-white" />
          </div>
          <span className="font-bold text-lg tracking-tight text-white">Statusy</span>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
          {navItems.map(({ to, icon: Icon, label, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                clsx(
                  'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-indigo-600/20 text-indigo-400'
                    : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800',
                )
              }
            >
              <Icon className="w-4 h-4 flex-shrink-0" />
              {label}
            </NavLink>
          ))}

          {user?.role === 'admin' && (
            <>
              <NavLink
                to="/status-pages"
                className={({ isActive }) =>
                  clsx(
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-indigo-600/20 text-indigo-400'
                      : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800',
                  )
                }
              >
                <Globe className="w-4 h-4 flex-shrink-0" />
                Status Pages
              </NavLink>
              <NavLink
                to="/users"
                className={({ isActive }) =>
                  clsx(
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-indigo-600/20 text-indigo-400'
                      : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800',
                  )
                }
              >
                <Users className="w-4 h-4 flex-shrink-0" />
                Users
              </NavLink>
              <NavLink
                to="/integrations"
                className={({ isActive }) =>
                  clsx(
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-indigo-600/20 text-indigo-400'
                      : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800',
                  )
                }
              >
                <Plug className="w-4 h-4 flex-shrink-0" />
                Integrations
              </NavLink>
            </>
          )}
        </nav>

        {/* User footer */}
        <div className="px-3 py-4 border-t border-gray-800">
          <div className="flex items-center gap-3 px-3 py-2 mb-1">
            <div className="w-7 h-7 rounded-full bg-indigo-600/30 flex items-center justify-center text-xs font-bold text-indigo-400 uppercase">
              {user?.username?.[0] ?? '?'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-200 truncate">{user?.username}</p>
              <p className="text-xs text-gray-500 capitalize">{user?.role}</p>
            </div>
          </div>
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            Sign out
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}
