import { useRef, useEffect, useState } from 'react'
import { FiChevronLeft, FiChevronRight } from 'react-icons/fi'
import type { Appointment } from '../types/api'

interface Props {
  appointments: Appointment[]
  timezone: string
  weekStart: 'monday' | 'sunday'
  onSelect: (appointment: Appointment) => void
}

const HOUR_HEIGHT = 64
const TOTAL_HEIGHT = 24 * HOUR_HEIGHT

// ── Date helpers ──────────────────────────────────────────────

function getWeekStartDate(date: Date, startDay: 'monday' | 'sunday'): Date {
  const d = new Date(date)
  const day = d.getDay()
  const diff = startDay === 'monday' ? (day === 0 ? -6 : 1 - day) : -day
  d.setDate(d.getDate() + diff)
  d.setHours(0, 0, 0, 0)
  return d
}

function getWeekDays(weekStart: Date): Date[] {
  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(weekStart)
    d.setDate(weekStart.getDate() + i)
    return d
  })
}

function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  )
}

function getDateStringInTz(utcString: string, timezone: string): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone: timezone }).format(new Date(utcString))
}

function formatDateISO(date: Date): string {
  return new Intl.DateTimeFormat('en-CA').format(date)
}

function getHourOffset(utcString: string, timezone: string): number {
  const parts = new Intl.DateTimeFormat('en-US', {
    hour: 'numeric',
    minute: 'numeric',
    hour12: false,
    timeZone: timezone,
  }).formatToParts(new Date(utcString))

  const hour = parseInt(parts.find(p => p.type === 'hour')?.value ?? '0')
  const minute = parseInt(parts.find(p => p.type === 'minute')?.value ?? '0')
  return (hour + minute / 60) * HOUR_HEIGHT
}

function getDurationHeight(startUtc: string, endUtc: string): number {
  const ms = new Date(endUtc).getTime() - new Date(startUtc).getTime()
  return Math.max((ms / 3_600_000) * HOUR_HEIGHT, 24)
}

function formatTime(utcString: string, timezone: string): string {
  return new Intl.DateTimeFormat('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
    timeZone: timezone,
  }).format(new Date(utcString))
}

function formatHour(h: number): string {
  if (h === 0) return '12 AM'
  if (h < 12) return `${h} AM`
  if (h === 12) return '12 PM'
  return `${h - 12} PM`
}

function getCurrentTimeOffset(): number {
  const now = new Date()
  return (now.getHours() + now.getMinutes() / 60) * HOUR_HEIGHT
}

function formatWeekRange(days: Date[]): string {
  const start = days[0]
  const end = days[6]
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' }
  if (start.getFullYear() !== end.getFullYear()) {
    return `${new Intl.DateTimeFormat('en-US', { ...opts, year: 'numeric' }).format(start)} – ${new Intl.DateTimeFormat('en-US', { ...opts, year: 'numeric' }).format(end)}`
  }
  if (start.getMonth() !== end.getMonth()) {
    return `${new Intl.DateTimeFormat('en-US', opts).format(start)} – ${new Intl.DateTimeFormat('en-US', opts).format(end)}`
  }
  return `${new Intl.DateTimeFormat('en-US', { month: 'long' }).format(start)} ${start.getDate()} – ${end.getDate()}, ${start.getFullYear()}`
}

// ── Appointment block ─────────────────────────────────────────

function AppointmentBlock({
  appointment,
  timezone,
  onSelect,
}: {
  appointment: Appointment
  timezone: string
  onSelect: (appointment: Appointment) => void
}) {
  const cancelled = appointment.status === 'cancelled'
  const top = getHourOffset(appointment.startTime, timezone)
  const height = getDurationHeight(appointment.startTime, appointment.endTime)

  return (
    <div
      onClick={() => onSelect(appointment)}
      style={{ top, height, left: 2, right: 2 }}
      className={`absolute rounded-sm px-2 py-1 overflow-hidden cursor-pointer ${
        cancelled
          ? 'bg-gray-100 border border-gray-200'
          : 'bg-black hover:bg-gray-800 transition-colors'
      }`}
    >
      <p
        className={`text-xs font-medium leading-tight truncate ${
          cancelled ? 'text-gray-400 line-through' : 'text-white'
        }`}
      >
        {appointment.title}
      </p>
      {height > 30 && (
        <p className="text-xs leading-tight truncate text-gray-400">
          {formatTime(appointment.startTime, timezone)} – {formatTime(appointment.endTime, timezone)}
        </p>
      )}
    </div>
  )
}

// ── Main calendar ─────────────────────────────────────────────

export default function WeeklyCalendar({ appointments, timezone, weekStart, onSelect }: Props) {
  const [weekStartDate, setWeekStartDate] = useState(() =>
    getWeekStartDate(new Date(), weekStart),
  )
  const scrollRef = useRef<HTMLDivElement>(null)
  const today = new Date()

  useEffect(() => {
    if (scrollRef.current) {
      const offset = getCurrentTimeOffset()
      scrollRef.current.scrollTop = Math.max(0, offset - 120)
    }
  }, [])

  const weekDays = getWeekDays(weekStartDate)

  function navigate(dir: -1 | 1) {
    setWeekStartDate(d => {
      const next = new Date(d)
      next.setDate(d.getDate() + dir * 7)
      return next
    })
  }

  function goToToday() {
    setWeekStartDate(getWeekStartDate(new Date(), weekStart))
  }

  return (
    <div className="flex flex-col h-full select-none">

      {/* Week navigation */}
      <div className="flex items-center gap-3 px-4 py-3 border-b border-gray-100 flex-shrink-0">
        <button
          onClick={() => navigate(-1)}
          className="p-1 hover:bg-gray-100 rounded-sm transition-colors"
        >
          <FiChevronLeft size={16} />
        </button>
        <button
          onClick={() => navigate(1)}
          className="p-1 hover:bg-gray-100 rounded-sm transition-colors"
        >
          <FiChevronRight size={16} />
        </button>
        <span className="text-sm font-medium text-black">{formatWeekRange(weekDays)}</span>
        <button
          onClick={goToToday}
          className="ml-auto text-xs font-medium px-3 py-1 border border-gray-300 rounded-sm hover:border-gray-400 transition-colors"
        >
          Today
        </button>
      </div>

      {/* Day headers */}
      <div className="flex border-b border-gray-100 flex-shrink-0">
        <div className="w-14 flex-shrink-0" />
        {weekDays.map((day, i) => {
          const isToday = isSameDay(day, today)
          return (
            <div key={i} className="flex-1 text-center py-2">
              <p className="text-xs text-gray-400 uppercase tracking-wide">
                {new Intl.DateTimeFormat('en-US', { weekday: 'short' }).format(day)}
              </p>
              <p
                className={`text-sm font-semibold mt-0.5 w-7 h-7 flex items-center justify-center mx-auto rounded-full ${
                  isToday ? 'bg-black text-white' : 'text-gray-700'
                }`}
              >
                {day.getDate()}
              </p>
            </div>
          )
        })}
      </div>

      {/* Scrollable grid */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto">
        <div className="flex" style={{ height: TOTAL_HEIGHT }}>

          {/* Time labels */}
          <div className="w-14 flex-shrink-0 relative">
            {Array.from({ length: 24 }, (_, hour) => (
              <div
                key={hour}
                style={{ top: hour * HOUR_HEIGHT - 8 }}
                className="absolute right-2 text-xs text-gray-400 text-right leading-none"
              >
                {hour === 0 ? '' : formatHour(hour)}
              </div>
            ))}
          </div>

          {/* Day columns */}
          {weekDays.map((day, colIdx) => {
            const dateStr = formatDateISO(day)
            const dayAppointments = appointments.filter(
              a => getDateStringInTz(a.startTime, timezone) === dateStr,
            )
            const isToday = isSameDay(day, today)

            return (
              <div
                key={colIdx}
                className={`flex-1 relative border-l border-gray-100 ${
                  isToday ? 'bg-gray-50/50' : ''
                }`}
              >
                {/* Hour lines */}
                {Array.from({ length: 24 }, (_, hour) => (
                  <div
                    key={hour}
                    style={{ top: hour * HOUR_HEIGHT }}
                    className="absolute w-full border-t border-gray-100"
                  />
                ))}

                {/* Current time indicator */}
                {isToday && (
                  <div
                    style={{ top: getCurrentTimeOffset() }}
                    className="absolute w-full z-10 flex items-center"
                  >
                    <div className="w-2 h-2 rounded-full bg-black -ml-1 flex-shrink-0" />
                    <div className="flex-1 border-t border-black" />
                  </div>
                )}

                {/* Appointments */}
                {dayAppointments.map(appt => (
                  <AppointmentBlock
                    key={appt.id}
                    appointment={appt}
                    timezone={timezone}
                    onSelect={onSelect}
                  />
                ))}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
