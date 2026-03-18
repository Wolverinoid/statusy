import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/contexts/AuthContext'
import Layout from '@/components/Layout'
import Login from '@/pages/Login'
import Setup from '@/pages/Setup'
import Dashboard from '@/pages/Dashboard'
import Monitors from '@/pages/Monitors'
import MonitorDetail from '@/pages/MonitorDetail'
import Notifications from '@/pages/Notifications'
import Users from '@/pages/Users'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-950">
        <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }
  return user ? <>{children}</> : <Navigate to="/login" replace />
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  return user?.role === 'admin' ? <>{children}</> : <Navigate to="/" replace />
}

// Checks /api/setup/status and redirects to /setup if needed.
function SetupGuard({ children }: { children: React.ReactNode }) {
  const [checked, setChecked] = useState(false)
  const [needsSetup, setNeedsSetup] = useState(false)

  useEffect(() => {
    fetch('/api/setup/status')
      .then((r) => r.json())
      .then((d) => {
        setNeedsSetup(d.needed === true)
        setChecked(true)
      })
      .catch(() => setChecked(true)) // if endpoint missing, proceed normally
  }, [])

  if (!checked) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-950">
        <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (needsSetup) {
    return <Navigate to="/setup" replace />
  }

  return <>{children}</>
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/setup" element={<Setup />} />
          <Route path="/login" element={<SetupGuard><Login /></SetupGuard>} />
          <Route
            path="/"
            element={
              <SetupGuard>
                <PrivateRoute>
                  <Layout />
                </PrivateRoute>
              </SetupGuard>
            }
          >
            <Route index element={<Dashboard />} />
            <Route path="monitors" element={<Monitors />} />
            <Route path="monitors/:id" element={<MonitorDetail />} />
            <Route path="notifications" element={<Notifications />} />
            <Route
              path="users"
              element={
                <AdminRoute>
                  <Users />
                </AdminRoute>
              }
            />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
