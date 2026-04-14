import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'
import { FiEye, FiEyeOff } from 'react-icons/fi'
import { authApi, ApiError } from '../api/client'
import { useAuth } from '../context/AuthContext'

type Tab = 'login' | 'register'
type FieldErrors = Record<string, string>

const ALL_TIMEZONES = (Intl as unknown as { supportedValuesOf(key: string): string[] }).supportedValuesOf('timeZone')
const DETECTED_TIMEZONE = Intl.DateTimeFormat().resolvedOptions().timeZone

const loginSchema = z.object({
  email: z.string().email('Enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
})

const registerSchema = z
  .object({
    firstName: z.string().min(1, 'First name is required'),
    lastName: z.string().min(1, 'Last name is required'),
    email: z.string().email('Enter a valid email address'),
    password: z.string().min(8, 'Password must be at least 8 characters'),
    confirmPassword: z.string().min(1, 'Please confirm your password'),
    timezone: z.string().min(1, 'Timezone is required'),
    weekStart: z.enum(['monday', 'sunday']),
  })
  .refine(data => data.password === data.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword'],
  })

function parseErrors(error: z.ZodError): FieldErrors {
  const flat = error.flatten().fieldErrors as Record<string, string[] | undefined>
  return Object.fromEntries(
    Object.entries(flat).map(([k, v]) => [k, v?.[0] ?? ''])
  )
}

export default function AuthPage() {
  const [tab, setTab] = useState<Tab>('login')
  const [loading, setLoading] = useState(false)
  const [loginErrors, setLoginErrors] = useState<FieldErrors>({})
  const [registerErrors, setRegisterErrors] = useState<FieldErrors>({})
  const { login } = useAuth()
  const navigate = useNavigate()

  const [loginForm, setLoginForm] = useState({ email: '', password: '' })
  const [registerForm, setRegisterForm] = useState({
    firstName: '',
    lastName: '',
    email: '',
    password: '',
    confirmPassword: '',
    timezone: DETECTED_TIMEZONE,
    weekStart: 'monday' as 'monday' | 'sunday',
  })

  function setLoginField<K extends keyof typeof loginForm>(key: K, value: string) {
    setLoginForm(f => ({ ...f, [key]: value }))
    setLoginErrors(e => ({ ...e, [key]: '' }))
  }

  function setRegisterField<K extends keyof typeof registerForm>(key: K, value: typeof registerForm[K]) {
    setRegisterForm(f => ({ ...f, [key]: value }))
    setRegisterErrors(e => ({ ...e, [key]: '' }))
  }

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    const result = loginSchema.safeParse(loginForm)
    if (!result.success) {
      setLoginErrors(parseErrors(result.error))
      return
    }
    setLoading(true)
    try {
      const res = await authApi.login(loginForm)
      login(res.token, res.userId)
      toast.success('Welcome back!')
      navigate('/dashboard', { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'An unexpected error occurred')
    } finally {
      setLoading(false)
    }
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault()
    const result = registerSchema.safeParse(registerForm)
    if (!result.success) {
      setRegisterErrors(parseErrors(result.error))
      return
    }
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
                onClick={() => { setTab(t); setLoginErrors({}); setRegisterErrors({}) }}
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
            <form onSubmit={handleLogin} className="space-y-3">
              <Field label="Email" error={loginErrors.email}>
                <input
                  type="email"
                  value={loginForm.email}
                  onChange={e => setLoginField('email', e.target.value)}
                  placeholder="you@example.com"
                  className={inputClass(!!loginErrors.email)}
                />
              </Field>
              <Field label="Password" error={loginErrors.password}>
                <PasswordInput
                  value={loginForm.password}
                  onChange={v => setLoginField('password', v)}
                  hasError={!!loginErrors.password}
                />
              </Field>
              <SubmitButton loading={loading} label="Log in" />
            </form>
          ) : (
            <form onSubmit={handleRegister} className="space-y-3">
              <div className="flex gap-3">
                <Field label="First name" error={registerErrors.firstName}>
                  <input
                    type="text"
                    value={registerForm.firstName}
                    onChange={e => setRegisterField('firstName', e.target.value)}
                    placeholder="John"
                    className={inputClass(!!registerErrors.firstName)}
                  />
                </Field>
                <Field label="Last name" error={registerErrors.lastName}>
                  <input
                    type="text"
                    value={registerForm.lastName}
                    onChange={e => setRegisterField('lastName', e.target.value)}
                    placeholder="Doe"
                    className={inputClass(!!registerErrors.lastName)}
                  />
                </Field>
              </div>
              <Field label="Email" error={registerErrors.email}>
                <input
                  type="email"
                  value={registerForm.email}
                  onChange={e => setRegisterField('email', e.target.value)}
                  placeholder="you@example.com"
                  className={inputClass(!!registerErrors.email)}
                />
              </Field>
              <Field label="Password" error={registerErrors.password}>
                <PasswordInput
                  value={registerForm.password}
                  onChange={v => setRegisterField('password', v)}
                  placeholder="Min. 8 characters"
                  hasError={!!registerErrors.password}
                />
              </Field>
              <Field label="Confirm password" error={registerErrors.confirmPassword}>
                <PasswordInput
                  value={registerForm.confirmPassword}
                  onChange={v => setRegisterField('confirmPassword', v)}
                  placeholder="Re-enter your password"
                  hasError={!!registerErrors.confirmPassword}
                />
              </Field>
              <Field label="Timezone" error={registerErrors.timezone}>
                <TimezoneSelect
                  value={registerForm.timezone}
                  onChange={tz => setRegisterField('timezone', tz)}
                  hasError={!!registerErrors.timezone}
                />
              </Field>
              <Field label="Week starts on">
                <div className="flex gap-2">
                  {(['monday', 'sunday'] as const).map(day => (
                    <button
                      key={day}
                      type="button"
                      onClick={() => setRegisterField('weekStart', day)}
                      className={`flex-1 py-2 text-sm font-medium rounded-sm border transition-colors capitalize ${
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

function Field({
  label,
  error,
  children,
}: {
  label: string
  error?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex-1">
      <label className="block text-xs font-medium text-gray-500 mb-1">{label}</label>
      {children}
      <div className="min-h-[12px] mt-0.5">
        {error && <p className="text-xs text-red-500 leading-none">{error}</p>}
      </div>
    </div>
  )
}

function PasswordInput({
  value,
  onChange,
  placeholder = '••••••••',
  hasError = false,
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  hasError?: boolean
}) {
  const [visible, setVisible] = useState(false)
  return (
    <div className="relative">
      <input
        type={visible ? 'text' : 'password'}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className={`${inputClass(hasError)} pr-10`}
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
  hasError = false,
}: {
  value: string
  onChange: (tz: string) => void
  hasError?: boolean
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
        className={inputClass(hasError)}
      />
      {open && (
        <div className="absolute z-20 w-full bg-white border border-gray-200 rounded-sm shadow-lg max-h-48 overflow-y-auto mt-1">
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
      className="w-full bg-black text-white text-sm font-medium py-2.5 rounded-sm hover:bg-gray-900 disabled:opacity-50 disabled:cursor-not-allowed transition-colors mt-2"
    >
      {loading ? 'Please wait...' : label}
    </button>
  )
}

function inputClass(hasError: boolean) {
  return `w-full border ${hasError ? 'border-red-400' : 'border-gray-300'} rounded-sm px-3 py-2 text-sm text-black placeholder-gray-400 focus:border-black transition-colors bg-white`
}
