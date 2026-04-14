import { FiX, FiCalendar, FiClock } from 'react-icons/fi'
import { toast } from 'sonner'
import { useCancelAppointment } from '../hooks/useAppointments'
import type { Appointment } from '../types/api'

interface Props {
  appointment: Appointment | null
  timezone: string
  onClose: () => void
  onCancelled: (updated: Appointment) => void
}

function formatDate(utcString: string, timezone: string): string {
  return new Intl.DateTimeFormat('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    timeZone: timezone,
  }).format(new Date(utcString))
}

function formatTime(utcString: string, timezone: string): string {
  return new Intl.DateTimeFormat('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
    timeZone: timezone,
  }).format(new Date(utcString))
}

export default function AppointmentDetailPanel({ appointment, timezone, onClose, onCancelled }: Props) {
  const { mutate: cancel, isPending } = useCancelAppointment()
  const open = appointment !== null

  function handleCancel() {
    if (!appointment) return
    toast('Cancel this appointment?', {
      action: {
        label: 'Confirm',
        onClick: () => cancel(appointment.id, {
          onSuccess: (data) => onCancelled(data.appointment),
        }),
      },
    })
  }

  return (
    <>
      {open && (
        <div className="fixed inset-0 z-40 bg-black/10" onClick={onClose} />
      )}
      <div
        className={`fixed right-0 top-0 h-full w-3/4 md:w-1/4 bg-white z-50 shadow-2xl border-l border-gray-100 flex flex-col transform transition-transform duration-300 ease-in-out ${
          open ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        {appointment && (
          <>
            <div className="flex items-start justify-between px-6 py-5 border-b border-gray-100">
              <div className="flex-1 pr-4">
                <p className={`text-xs font-medium mb-1 uppercase tracking-wide ${
                  appointment.status === 'completed' ? 'text-blue-600' :
                  appointment.status === 'cancelled' ? 'text-gray-400' :
                  'text-gray-400'
                }`}>
                  {appointment.status === 'cancelled' ? 'Cancelled' : appointment.status === 'completed' ? 'Completed' : 'Scheduled'}
                </p>
                <h2
                  className={`text-base font-semibold leading-snug ${
                    appointment.status === 'cancelled' ? 'text-gray-400 line-through' : 'text-black'
                  }`}
                >
                  {appointment.title}
                </h2>
              </div>
              <button
                onClick={onClose}
                className="text-gray-400 hover:text-black transition-colors mt-0.5 flex-shrink-0"
              >
                <FiX size={18} />
              </button>
            </div>

            <div className="flex-1 px-6 py-5 space-y-4 overflow-y-auto">
              <div className="flex items-start gap-3">
                <FiCalendar size={14} className="text-gray-400 mt-0.5 flex-shrink-0" />
                <p className="text-sm text-gray-700">{formatDate(appointment.startTime, timezone)}</p>
              </div>
              <div className="flex items-start gap-3">
                <FiClock size={14} className="text-gray-400 mt-0.5 flex-shrink-0" />
                <p className="text-sm text-gray-700">
                  {formatTime(appointment.startTime, timezone)} – {formatTime(appointment.endTime, timezone)}
                </p>
              </div>
              {appointment.description && (
                <div className="pt-2 border-t border-gray-100">
                  <p className="text-xs font-medium text-gray-400 mb-1.5">Description</p>
                  <p className="text-sm text-gray-700 leading-relaxed">{appointment.description}</p>
                </div>
              )}
              {appointment.recurrenceGroupId && (
                <span className="inline-block text-xs font-medium text-gray-500 bg-gray-100 px-2 py-0.5 rounded-sm">
                  Recurring
                </span>
              )}
            </div>

            {appointment.status === 'scheduled' && (
              <div className="px-6 py-5 border-t border-gray-100">
                <button
                  onClick={handleCancel}
                  disabled={isPending}
                  className="w-full py-2.5 text-sm font-medium border border-red-200 text-red-500 rounded-sm hover:bg-red-50 transition-colors disabled:opacity-50"
                >
                  Cancel appointment
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </>
  )
}
