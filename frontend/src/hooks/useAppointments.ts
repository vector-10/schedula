import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { appointmentsApi } from '../api/client'
import type { CreateAppointmentRequest } from '../types/api'

const APPOINTMENTS_KEY = ['appointments']

export function useAppointments() {
  return useQuery({
    queryKey: APPOINTMENTS_KEY,
    queryFn: () => appointmentsApi.list(),
  })
}

export function useCreateAppointment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateAppointmentRequest) => appointmentsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: APPOINTMENTS_KEY })
      toast.success('Appointment booked')
    },
    onError: (err: Error) => {
      toast.error(err.message)
    },
  })
}

export function useCancelAppointment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (appointmentId: string) => appointmentsApi.cancel(appointmentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: APPOINTMENTS_KEY })
      toast.success('Appointment cancelled')
    },
    onError: (err: Error) => {
      toast.error(err.message)
    },
  })
}
