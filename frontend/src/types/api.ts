export interface Appointment {
  id: string
  userId: string
  title: string
  description: string
  startTime: string
  endTime: string
  status: 'scheduled' | 'completed' | 'cancelled'
  recurrenceGroupId: string
  createdAt: string
  updatedAt: string
}

export interface RegisterRequest {
  email: string
  password: string
  timezone: string
  weekStart: 'monday' | 'sunday'
  firstName: string
  lastName: string
}

export interface UserProfile {
  id: string
  email: string
  firstName: string
  lastName: string
  timezone: string
  weekStart: 'monday' | 'sunday'
}

export interface RegisterResponse {
  token: string
  userId: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  userId: string
}

export interface CreateAppointmentRequest {
  title: string
  description?: string
  startTime: string
  endTime: string
  idempotencyKey: string
  recurrenceRule?: string
  recurrenceEndDate?: string
}

export interface CreateAppointmentResponse {
  appointments: Appointment[]
}

export interface GetAppointmentsResponse {
  appointments: Appointment[]
  userTimezone: string
  weekStart: 'monday' | 'sunday'
}

export interface CancelAppointmentResponse {
  appointment: Appointment
}

export interface ApiError {
  code: number
  message: string
}
