import { useState, useEffect } from 'react'
import { v4 as uuidv4 } from 'uuid'
import { FiX } from 'react-icons/fi'
import { useCreateAppointment } from '../hooks/useAppointments'

interface Props {
  isOpen: boolean
  onClose: () => void
}

function today(): string {
  return new Date().toISOString().split('T')[0]
}

export default function AppointmentModal({ isOpen, onClose }: Props) {
  const [form, setForm] = useState({
    title: '',
    description: '',
    date: today(),
    startTime: '09:00',
    endTime: '10:00',
    isRecurring: false,
    recurrenceEndDate: '',
  })
  const [timeError, setTimeError] = useState('')

  const { mutate: create, isPending } = useCreateAppointment()

  useEffect(() => {
    if (isOpen) {
      setForm({
        title: '',
        description: '',
        date: today(),
        startTime: '09:00',
        endTime: '10:00',
        isRecurring: false,
        recurrenceEndDate: '',
      })
      setTimeError('')
    }
  }, [isOpen])

  function handleStartTimeChange(value: string) {
    setForm(f => {
      const [sh, sm] = value.split(':').map(Number)
      const endTotalMins = sh * 60 + sm + 60
      const eh = Math.floor(endTotalMins / 60) % 24
      const em = endTotalMins % 60
      const autoEnd = `${String(eh).padStart(2, '0')}:${String(em).padStart(2, '0')}`
      return { ...f, startTime: value, endTime: autoEnd }
    })
    setTimeError('')
  }

  function handleEndTimeChange(value: string) {
    setForm(f => ({ ...f, endTime: value }))
    setTimeError('')
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    if (form.endTime <= form.startTime) {
      setTimeError('End time must be after start time')
      return
    }

    if (form.isRecurring && !form.recurrenceEndDate) {
      setTimeError('Recurrence end date is required')
      return
    }

    const startTime = new Date(`${form.date}T${form.startTime}:00`).toISOString()
    const endTime = new Date(`${form.date}T${form.endTime}:00`).toISOString()

    create(
      {
        title: form.title,
        description: form.description || undefined,
        startTime,
        endTime,
        idempotencyKey: uuidv4(),
        ...(form.isRecurring && {
          recurrenceRule: 'WEEKLY',
          recurrenceEndDate: new Date(`${form.recurrenceEndDate}T00:00:00`).toISOString(),
        }),
      },
      { onSuccess: onClose },
    )
  }

  if (!isOpen) return null

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="bg-white w-full max-w-md rounded-sm shadow-xl">
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="text-base font-semibold text-black">New appointment</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-black transition-colors"
          >
            <FiX size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="px-6 py-5 space-y-4">
          <Field label="Title">
            <input
              type="text"
              required
              value={form.title}
              onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
              placeholder="e.g. Team standup"
              className={inputClass}
            />
          </Field>

          <Field label="Description">
            <textarea
              value={form.description}
              onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
              placeholder="Optional"
              rows={2}
              className={`${inputClass} resize-none`}
            />
          </Field>

          <Field label="Date">
            <input
              type="date"
              required
              value={form.date}
              min={today()}
              onChange={e => setForm(f => ({ ...f, date: e.target.value }))}
              className={inputClass}
            />
          </Field>

          <div className="flex gap-3">
            <Field label="Start time">
              <input
                type="time"
                required
                value={form.startTime}
                onChange={e => handleStartTimeChange(e.target.value)}
                className={inputClass}
              />
            </Field>
            <Field label="End time">
              <input
                type="time"
                required
                value={form.endTime}
                onChange={e => handleEndTimeChange(e.target.value)}
                className={inputClass}
              />
            </Field>
          </div>

          {timeError && (
            <p className="text-xs text-red-500">{timeError}</p>
          )}

          <div className="flex items-center justify-between py-1">
            <span className="text-sm text-gray-700 font-medium">Repeat weekly</span>
            <button
              type="button"
              onClick={() => setForm(f => ({ ...f, isRecurring: !f.isRecurring, recurrenceEndDate: '' }))}
              className={`relative w-10 h-5 rounded-full transition-colors ${
                form.isRecurring ? 'bg-black' : 'bg-gray-300'
              }`}
            >
              <span
                className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow-sm transition-transform duration-200 ${
                  form.isRecurring ? 'translate-x-[2px]' : 'translate-x-[-17px]'
                }`}
              />
            </button>
          </div>

          {form.isRecurring && (
            <Field label="Repeat until (max 4 occurrences)">
              <input
                type="date"
                required
                value={form.recurrenceEndDate}
                min={form.date}
                onChange={e => setForm(f => ({ ...f, recurrenceEndDate: e.target.value }))}
                className={inputClass}
              />
            </Field>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2.5 text-sm font-medium border border-gray-300 rounded-sm text-gray-700 hover:border-gray-400 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isPending}
              className="flex-1 py-2.5 text-sm font-medium bg-black text-white rounded-sm hover:bg-gray-900 disabled:opacity-50 transition-colors"
            >
              {isPending ? 'Booking...' : 'Book appointment'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex-1">
      <label className="block text-xs font-medium text-gray-500 mb-1">{label}</label>
      {children}
    </div>
  )
}

const inputClass =
  'w-full border border-gray-300 rounded-sm px-3 py-2 text-sm text-black placeholder-gray-400 focus:border-black transition-colors bg-white'
