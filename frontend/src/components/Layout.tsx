import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { useTheme } from '@/contexts/ThemeContext'
import {
  LayoutDashboard, Activity, Bell, Users, LogOut, Radio, Globe, Plug, Sun, Moon,
} from 'lucide-react'
import clsx from 'clsx'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard', end: true },
  { to: '/monitors', icon: Activity, label: 'Monitors' },
  { to: '/notifications', icon: Bell, label: 'Notifications' },
]

const adminItems = [
  { to: '/status-pages', icon: Globe, label: 'Status Pages' },
  { to: '/users', icon: Users, label: 'Users' },
  { to: '/integrations', icon: Plug, label: 'Integrations' },
]

export default function Layout() {
  const { user, logout } = useAuth()
  const { theme, toggle } = useTheme()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div className="flex h-screen overflow-hidden bg-gray-950 dark:bg-gray-950 bg-slate-50">
      {/* Sidebar */}
      <aside className="w-56 flex-shrink-0 flex flex-col border-r border-gray-800/50 dark:border-gray-800/50 border-slate-200 bg-gray-950 dark:bg-gray-950 bg-white relative">
        {/* Subtle vertical glow line on right edge */}
        <div className="absolute right-0 top-0 bottom-0 w-px bg-gradient-to-b from-transparent via-cyan-500/20 to-transparent pointer-events-none dark:block hidden" />

        {/* Logo */}
        <div className="flex items-center gap-3 px-5 py-5 border-b border-gray-800/50 dark:border-gray-800/50 border-slate-200">
          <div className="relative w-8 h-8 flex items-center justify-center">
            <div className="absolute inset-0 bg-cyan-500/20 rounded-lg blur-sm dark:block hidden" />
            <div className="relative w-8 h-8 dark:bg-gray-900 bg-cyan-50 border dark:border-cyan-500/40 border-cyan-300/60 rounded-lg flex items-center justify-center">
              <Radio className="w-4 h-4 text-cyan-500" />
            </div>
          </div>
          <div>
            <span className="font-bold text-base tracking-tight dark:text-white text-gray-900">Statusy</span>
            <div className="flex items-center gap-1 mt-0.5">
              <span className="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse" />
              <span className="text-[10px] text-cyan-600 font-mono uppercase tracking-widest">live</span>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
          {navItems.map(({ to, icon: Icon, label, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                clsx(isActive ? 'nav-item-active' : 'nav-item-inactive')
              }
            >
              {({ isActive }) => (
                <>
                  <Icon className={clsx('w-4 h-4 flex-shrink-0', isActive ? 'text-cyan-400' : 'text-gray-600')} />
                  {label}
                </>
              )}
            </NavLink>
          ))}

          {user?.role === 'admin' && (
            <>
              <div className="pt-3 pb-1 px-3">
                <span className="text-[10px] font-mono text-gray-700 uppercase tracking-widest">Admin</span>
              </div>
              {adminItems.map(({ to, icon: Icon, label }) => (
                <NavLink
                  key={to}
                  to={to}
                  className={({ isActive }) =>
                    clsx(isActive ? 'nav-item-active' : 'nav-item-inactive')
                  }
                >
                  {({ isActive }) => (
                    <>
                      <Icon className={clsx('w-4 h-4 flex-shrink-0', isActive ? 'text-cyan-400' : 'text-gray-600')} />
                      {label}
                    </>
                  )}
                </NavLink>
              ))}
            </>
          )}
        </nav>

        {/* User footer */}
        <div className="px-3 py-4 border-t dark:border-gray-800/50 border-slate-200">
          <div className="flex items-center gap-2.5 px-2 py-2 mb-1 rounded-lg">
            <div className="w-7 h-7 rounded-lg dark:bg-gray-800 bg-slate-100 dark:border-gray-700/60 border-slate-300 border flex items-center justify-center text-xs font-bold text-cyan-500 font-mono uppercase flex-shrink-0">
              {user?.username?.[0] ?? '?'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium dark:text-gray-300 text-gray-700 truncate">{user?.username}</p>
              <p className="text-[10px] dark:text-gray-600 text-slate-400 font-mono uppercase tracking-wider">{user?.role}</p>
            </div>
            {/* Theme toggle */}
            <button
              onClick={toggle}
              title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
              className="p-1.5 rounded-lg dark:text-gray-600 text-slate-400 dark:hover:text-cyan-400 hover:text-cyan-600 dark:hover:bg-gray-800 hover:bg-slate-100 transition-all duration-150 flex-shrink-0"
            >
              {theme === 'dark' ? <Sun className="w-3.5 h-3.5" /> : <Moon className="w-3.5 h-3.5" />}
            </button>
          </div>
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm dark:text-gray-600 text-slate-500 hover:text-red-500 dark:hover:bg-red-500/5 hover:bg-red-50 transition-all duration-150"
          >
            <LogOut className="w-4 h-4" />
            Sign out
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto dark:bg-gray-950 bg-slate-50">
        <Outlet />
      </main>
    </div>
  )
}
