import { useQuery } from '@tanstack/react-query'
import { authApi } from '../api/client'
import { useAuth } from '../context/AuthContext'

export const PROFILE_KEY = ['profile']

export function useProfile() {
  const { isAuthenticated } = useAuth()

  return useQuery({
    queryKey: PROFILE_KEY,
    queryFn: () => authApi.getProfile(),
    enabled: isAuthenticated,
  })
}
