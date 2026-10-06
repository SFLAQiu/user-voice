import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  token: string | null
  username: string | null
  role: string | null
  permissions: Record<string, string[]> | null
  setAuth: (token: string, username: string, role: string) => void
  setPermissions: (permissions: Record<string, string[]>) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      username: null,
      role: null,
      permissions: null,
      setAuth: (token, username, role) => set({ token, username, role }),
      setPermissions: (permissions) => set({ permissions }),
      logout: () => set({ token: null, username: null, role: null, permissions: null }),
    }),
    { name: 'auth-storage' },
  ),
)
