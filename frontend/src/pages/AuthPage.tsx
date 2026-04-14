import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { FiEye, FiEyeOff } from 'react-icons/fi'
import { authApi, ApiError } from '../api/client'
import { useAuth } from '../context/AuthContext'

type Tab = 'login' | 'register'

const ALL_TIMEZONES = (Intl as unknown as { supportedValuesOf(key: string): string[] }).supportedValuesOf('timeZone')
const DETECTED_TIMEZONE = Intl.DateTimeFormat().resolvedOptions().timeZone

export default function AuthPage() {
  const [tab, setTab] = useState<Tab>('login')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  const [loginForm, setLoginForm] = useState({ email: '', password: '' })
  const [registerForm, setRegisterForm] = useState({
    firstName: '',
    lastName: '',
    email: '',
    password: '',
    timezone: DETECTED_TIMEZONE,
    weekStart: 'monday' as 'monday' | 'sunday',
  })

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    try {
      const res = await authApi.login(loginForm)
      login(res.token, res.userId)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'An unexpected error occurred')
    } finally {
      setLoading(false)
    }
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    try {
      const res = await authApi.register(registerForm)
      login(res.token, res.userId)
      toast.success('Account created successfully!')
      navigate('/dashboard', { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'An unexpected error occurred')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-white flex">
      <div className="hidden lg:flex w-1/2 bg-black flex-col justify-between p-12">
        <span className="text-white text-xl font-semibold tracking-tight">Schedula</span>
        <div>
          <p className="text-white text-4xl font-bold leading-tight">
            Schedule with <br /> precision.
          </p>
          <p className="text-gray-400 mt-4 text-sm leading-relaxed">
            Book, manage, and track appointments <br /> without the conflict.
          </p>
        </div>
        <p className="text-gray-600 text-xs">© {new Date().getFullYear()} Schedula</p>
      </div>

      <div className="flex-1 flex items-center justify-center px-6">
        <div className="w-full max-w-sm">
          <h1 className="text-2xl font-bold text-black mb-1 lg:hidden">Schedula</h1>
          <h2 className="text-xl font-semibold text-black mb-6">
            {tab === 'login' ? 'Welcome back' : 'Create an account'}
          </h2>

          <div className="flex border-b border-gray-200 mb-6">
            {(['login', 'register'] as Tab[]).map(t => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`pb-2 mr-6 text-sm font-medium transition-colors capitalize ${
                  tab === t
                    ? 'text-black border-b-2 border-black'
                    : 'text-gray-400 hover:text-gray-600'
                }`}
              >
                {t === 'login' ? 'Log in' : 'Register'}
              </button>
            ))}
          </div>

          {tab === 'login' ? (
            <form onSubmit={handleLogin} className="space-y-4">
              <Field label="Email">
                <input
                  type="email"
                  required
                  value={loginForm.email}
                  onChange={e => setLoginForm(f => ({ ...f, email: e.target.value }))}
                  placeholder="you@example.com"
                  className={inputClass}
                />
              </Field>
              <Field label="Password">
                <PasswordInput
                  value={loginForm.password}
                  onChange={v => setLoginForm(f => ({ ...f, password: v }))}
                />
              </Field>
              <SubmitButton loading={loading} label="Log in" />
            </form>
          ) : (
            <form onSubmit={handleRegister} className="space-y-4">
              <div className="flex gap-3">
                <Field label="First name">
                  <input
                    type="text"
                    required
                    value={registerForm.firstName}
                    onChange={e => setRegisterForm(f => ({ ...f, firstName: e.target.value }))}
                    placeholder="John"
                    className={inputClass}
                  />
                </Field>
                <Field label="Last name">
                  <input
                    type="text"
                    required
                    value={registerForm.lastName}
                    onChange={e => setRegisterForm(f => ({ ...f, lastName: e.target.value }))}
                    placeholder="Doe"
                    className={inputClass}
                  />
                </Field>
              </div>
              <Field label="Email">
                <input
                  type="email"
                  required
                  value={registerForm.email}
                  onChange={e => setRegisterForm(f => ({ ...f, email: e.target.value }))}
                  placeholder="you@example.com"
                  className={inputClass}
                />
              </Field>
              <Field label="Password">
                <PasswordInput
                  value={registerForm.password}
                  onChange={v => setRegisterForm(f => ({ ...f, password: v }))}
                  minLength={8}
                  placeholder="Min. 8 characters"
                />
              </Field>
              <Field label="Timezone">
                <TimezoneSelect
                  value={registerForm.timezone}
                  onChange={tz => setRegisterForm(f => ({ ...f, timezone: tz }))}
                />
              </Field>
              <Field label="Week starts on">
                <div className="flex gap-2">
                  {(['monday', 'sunday'] as const).map(day => (
                    <button
                      key={day}
                      type="button"
                      onClick={() => setRegisterForm(f => ({ ...f, weekStart: day }))}
                      className={`flex-1 py-2 text-sm font-medium rounded-xs border transition-colors capitalize ${
                        registerForm.weekStart === day
                          ? 'bg-black text-white border-black'
                          : 'bg-white text-gray-500 border-gray-300 hover:border-gray-400'
                      }`}
                    >
                      {day}
                    </button>
                  ))}
                </div>
              </Field>
              <SubmitButton loading={loading} label="Create account" />
            </form>
          )}
        </div>
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-500 mb-1">{label}</label>
      {children}
    </div>
  )
}

function PasswordInput({
  value,
  onChange,
  minLength,
  placeholder = '••••••••',
}: {
  value: string
  onChange: (v: string) => void
  minLength?: number
  placeholder?: string
}) {
  const [visible, setVisible] = useState(false)
  return (
    <div className="relative">
      <input
        type={visible ? 'text' : 'password'}
        required
        minLength={minLength}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className={`${inputClass} pr-10`}
      />
      <button
        type="button"
        onClick={() => setVisible(v => !v)}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
      >
        {visible ? <FiEyeOff size={15} /> : <FiEye size={15} />}
      </button>
    </div>
  )
}

function TimezoneSelect({
  value,
  onChange,
}: {
  value: string
  onChange: (tz: string) => void
}) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const filtered = query.trim()
    ? ALL_TIMEZONES.filter((tz: string) => tz.toLowerCase().includes(query.toLowerCase()))
    : ALL_TIMEZONES

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
        setQuery('')
      }
    }
    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [])

  return (
    <div ref={containerRef} className="relative">
      <input
        type="text"
        value={open ? query : value}
        onChange={e => setQuery(e.target.value)}
        onFocus={() => setOpen(true)}
        placeholder="Search timezone..."
        className={inputClass}
      />
      {open && (
        <div className="absolute z-20 w-full bg-white border border-gray-200 rounded-xs shadow-lg max-h-48 overflow-y-auto mt-1">
          {filtered.length === 0 ? (
            <p className="px-3 py-2 text-sm text-gray-400">No results</p>
          ) : (
            filtered.slice(0, 80).map((tz: string) => (
              <button
                key={tz}
                type="button"
                onMouseDown={() => {
                  onChange(tz)
                  setOpen(false)
                  setQuery('')
                }}
                className={`w-full text-left px-3 py-2 text-sm transition-colors hover:bg-gray-50 ${
                  tz === value ? 'bg-gray-100 font-medium' : 'text-black'
                }`}
              >
                {tz}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  )
}

function SubmitButton({ loading, label }: { loading: boolean; label: string }) {
  return (
    <button
      type="submit"
      disabled={loading}
      className="w-full bg-black text-white text-sm font-medium py-2.5 rounded-xs hover:bg-gray-900 disabled:opacity-50 disabled:cursor-not-allowed transition-colors mt-2"
    >
      {loading ? 'Please wait...' : label}
    </button>
  )
}

const inputClass =
  'w-full border border-gray-300 rounded-xs px-3 py-2 text-sm text-black placeholder-gray-400 focus:border-black transition-colors bg-white'
