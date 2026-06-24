import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import type { User } from '@/lib/types';
import { api, getToken, setToken } from '@/lib/api';

interface AuthCtx {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    if (!getToken()) {
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      const u = await api.me();
      setUser(u);
    } catch {
      setToken(null);
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const login = useCallback(async (email: string, password: string) => {
    const res = await api.login({ email, password });
    setToken(res.token);
    setUser(res.user);
  }, []);

  const register = useCallback(async (username: string, email: string, password: string) => {
    const res = await api.register({ username, email, password });
    setToken(res.token);
    setUser(res.user);
  }, []);

  const logout = useCallback(async () => {
    await api.logout();
    setToken(null);
    setUser(null);
  }, []);

  const value = useMemo(
    () => ({ user, loading, login, register, logout, refresh }),
    [user, loading, login, register, logout, refresh],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useAuth() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
