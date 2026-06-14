const TOKEN_KEY = 'meta_org.token'
const USER_KEY = 'meta_org.user'
const LEGACY_TOKEN_KEY = 'harness_token'
const LEGACY_USER_KEY = 'harness_user'

export interface SessionUser {
  id: string
  type: string
}

function migrateLegacySession(): void {
  if (typeof window === 'undefined') return
  const token = localStorage.getItem(TOKEN_KEY) || localStorage.getItem(LEGACY_TOKEN_KEY)
  const user = localStorage.getItem(USER_KEY) || localStorage.getItem(LEGACY_USER_KEY)
  if (token) localStorage.setItem(TOKEN_KEY, token)
  if (user) localStorage.setItem(USER_KEY, user)
  localStorage.removeItem(LEGACY_TOKEN_KEY)
  localStorage.removeItem(LEGACY_USER_KEY)
}

export function setSession(token: string, userId: string, userType: string): void {
  if (typeof window === 'undefined') return
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify({ id: userId, type: userType }))
}

export function getToken(): string | null {
  if (typeof window === 'undefined') return null
  migrateLegacySession()
  return localStorage.getItem(TOKEN_KEY)
}

export function getSessionUser(): SessionUser | null {
  if (typeof window === 'undefined') return null
  migrateLegacySession()
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null

  try {
    return JSON.parse(raw) as SessionUser
  } catch {
    clearSession()
    return null
  }
}

export function clearSession(): void {
  if (typeof window === 'undefined') return
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  localStorage.removeItem(LEGACY_TOKEN_KEY)
  localStorage.removeItem(LEGACY_USER_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}
