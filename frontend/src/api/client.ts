import type {
  CancelAppointmentResponse,
  CreateAppointmentRequest,
  CreateAppointmentResponse,
  GetAppointmentsResponse,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  UserProfile,
} from '../types/api'

const BASE_URL = '/v1'

export class ApiError extends Error {
  constructor(
    public readonly message: string,
    public readonly status: number,
    public readonly grpcCode?: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }

  get isUnauthenticated() {
    return this.status === 401
  }

  get isConflict() {
    return this.status === 409
  }

  get isValidation() {
    return this.status === 400
  }

  get isServerError() {
    return this.status >= 500
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options?.headers as Record<string, string>),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  let res: Response

  try {
    res = await fetch(`${BASE_URL}${path}`, { ...options, headers })
  } catch {
    throw new ApiError(
      'Unable to reach the server. Please check your connection.',
      0,
    )
  }

  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const message = body?.message ?? genericMessage(res.status)
    const grpcCode = body?.code as number | undefined

    if (res.status === 401) {
      window.dispatchEvent(new CustomEvent('auth:expired'))
    }

    throw new ApiError(message, res.status, grpcCode)
  }

  return res.json()
}

function genericMessage(status: number): string {
  switch (status) {
    case 400: return 'Invalid request. Please check your input.'
    case 401: return 'Your session has expired. Please log in again.'
    case 404: return 'Not found.'
    case 409: return 'This request conflicts with an existing record.'
    case 500: return 'Something went wrong on our end. Please try again.'
    default:  return 'An unexpected error occurred.'
  }
}

export const authApi = {
  register: (data: RegisterRequest) =>
    request<RegisterResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  login: (data: LoginRequest) =>
    request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  getProfile: () => request<UserProfile>('/auth/profile'),
}

export const appointmentsApi = {
  list: () => request<GetAppointmentsResponse>('/appointments'),

  create: (data: CreateAppointmentRequest) =>
    request<CreateAppointmentResponse>('/appointments', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  cancel: (appointmentId: string) =>
    request<CancelAppointmentResponse>(`/appointments/${appointmentId}/cancel`, {
      method: 'POST',
      body: JSON.stringify({}),
    }),
}
