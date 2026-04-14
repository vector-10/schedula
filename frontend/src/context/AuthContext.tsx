import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from 'react'
import { toast } from 'sonner'

interface AuthState {
  token: string | null
  userId: string | null
}

interface AuthContextValue extends AuthState {
  login: (token: string, userId: string) => void
  logout: () => void
  isAuthenticated: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [auth, setAuth] = useState<AuthState>(() => ({
    token: localStorage.getItem('token'),
    userId: localStorage.getItem('userId'),
  }))

  const login = useCallback((token: string, userId: string) => {
    localStorage.setItem('token', token)
    localStorage.setItem('userId', userId)
    setAuth({ token, userId })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('userId')
    setAuth({ token: null, userId: null })
  }, [])

  useEffect(() => {
    function onExpired() {
      logout()
      toast.error('Your session has expired. Please log in again.')
    }

    window.addEventListener('auth:expired', onExpired)
    return () => window.removeEventListener('auth:expired', onExpired)
  }, [logout])

  return (
    <AuthContext.Provider
      value={{
        ...auth,
        login,
        logout,
        isAuthenticated: !!auth.token,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
