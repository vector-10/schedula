import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { FiPlus, FiLogOut } from 'react-icons/fi'
import { useAuth } from '../context/AuthContext'
import { useProfile } from '../hooks/useProfile'
import { useAppointments } from '../hooks/useAppointments'
import WeeklyCalendar from '../components/WeeklyCalendar'
import AppointmentModal from '../components/AppointmentModal'

export default function DashboardPage() {
  const [isModalOpen, setIsModalOpen] = useState(false)
  const { logout } = useAuth()
  const navigate = useNavigate()

  const { data: profile } = useProfile()
  const { data, isLoading } = useAppointments()

  const timezone = data?.userTimezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone
  const weekStart = data?.weekStart ?? 'monday'
  const appointments = data?.appointments ?? []

  function handleLogout() {
    logout()
    navigate('/auth', { replace: true })
  }

  return (
    <div className="h-screen flex flex-col bg-white overflow-hidden">

      {/* Navbar */}
      <header className="h-14 border-b border-gray-200 flex items-center justify-between px-6 flex-shrink-0">
        <span className="text-base font-semibold text-black tracking-tight">Schedula</span>
        <div className="flex items-center gap-5">
          {profile && (
            <span className="text-sm text-gray-600">
              {profile.firstName} {profile.lastName}
            </span>
          )}
          <button
            onClick={handleLogout}
            className="flex items-center gap-1.5 text-sm text-gray-500 hover:text-black transition-colors"
          >
            <FiLogOut size={14} />
            Log out
          </button>
        </div>
      </header>

      {/* Calendar */}
      <main className="flex-1 overflow-hidden relative">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <p className="text-sm text-gray-400">Loading appointments...</p>
          </div>
        ) : (
          <WeeklyCalendar
            appointments={appointments}
            timezone={timezone}
            weekStart={weekStart}
          />
        )}

        {/* FAB */}
        <div className="fixed bottom-8 right-8 z-40 flex flex-col items-end gap-2">
          <span className="text-sm font-medium text-gray-700">
            Create appointment
          </span>
          <button
            onClick={() => setIsModalOpen(true)}
            className="w-12 h-12 bg-black text-white rounded-full flex items-center justify-center shadow-lg hover:bg-gray-900 transition-colors"
          >
            <FiPlus size={22} />
          </button>
        </div>
      </main>

      <AppointmentModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </div>
  )
}
