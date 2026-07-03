import { useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { LogoMark } from '@/components/Logo';
import { Spinner } from '@/components/ui';
import { useAuth } from '@/context/AuthContext';
import { useToast } from '@/context/ToastContext';
import { ApiError } from '@/lib/api';

export function AuthPage({ mode }: { mode: 'login' | 'register' }) {
  const isLogin = mode === 'login';
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const toast = useToast();
  const from = (location.state as { from?: string } | null)?.from ?? '/';

  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      if (isLogin) await login(email, password);
      else await register(username, email, password);
      toast(isLogin ? 'Welcome back!' : 'Account forged — welcome to IndieForge', 'success');
      navigate(from, { replace: true });
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Something went wrong', 'error');
    } finally {
      setBusy(false);
    }
  };


  return (
    <div className="container-page flex min-h-[calc(100vh-4rem)] items-center justify-center py-12">
      <div className="w-full max-w-md">
        <div className="mb-6 flex flex-col items-center text-center">
          <LogoMark className="h-12 w-12" />
          <h1 className="mt-4 text-2xl font-700 text-mist-50">
            {isLogin ? 'Sign in to IndieForge' : 'Create your account'}
          </h1>
          <p className="mt-1 text-sm text-mist-400">
            {isLogin ? 'Forge new worlds and play indie games.' : 'Upload games, build a following, and play.'}
          </p>
        </div>

        <form onSubmit={submit} className="card space-y-4 p-6">
          {!isLogin && (
            <div>
              <label htmlFor="username" className="label">
                Username
              </label>
              <input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="input"
                placeholder="pixelsmith"
                autoComplete="username"
                required
              />
            </div>
          )}
          <div>
            <label htmlFor="email" className="label">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="input"
              placeholder="you@example.com"
              autoComplete="email"
              required
            />
          </div>
          <div>
            <label htmlFor="password" className="label">
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input"
              placeholder="••••••••"
              autoComplete={isLogin ? 'current-password' : 'new-password'}
              required
            />
          </div>

          <button type="submit" disabled={busy} className="btn btn-primary btn-lg w-full">
            {busy ? <Spinner /> : isLogin ? 'Sign in' : 'Create account'}
          </button>

        </form>

        <p className="mt-5 text-center text-sm text-mist-400">
          {isLogin ? "Don't have an account? " : 'Already have an account? '}
          <Link
            to={isLogin ? '/register' : '/login'}
            state={{ from }}
            className="font-600 text-ember-400 hover:text-ember-500"
          >
            {isLogin ? 'Sign up' : 'Sign in'}
          </Link>
        </p>
      </div>
    </div>
  );
}
