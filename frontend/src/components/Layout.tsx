import { useState } from 'react';
import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom';
import { Search, Library, LayoutDashboard, Plus, LogOut, Menu, X, Hammer, ShieldAlert, Sliders } from 'lucide-react';
import { Logo, LogoMark } from './Logo';
import { useAuth } from '@/context/AuthContext';

function NavItem({ to, icon: Icon, label }: { to: string; icon: typeof Library; label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors duration-200 ${
          isActive ? 'bg-iron-800 text-ember-400' : 'text-mist-300 hover:bg-iron-800/60 hover:text-mist-50'
        }`
      }
    >
      <Icon className="h-4 w-4" />
      {label}
    </NavLink>
  );
}

export function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [q, setQ] = useState('');
  const [mobileOpen, setMobileOpen] = useState(false);

  const submitSearch = (e: React.FormEvent) => {
    e.preventDefault();
    navigate(`/?search=${encodeURIComponent(q.trim())}`);
    setMobileOpen(false);
  };

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-30 border-b border-iron-700/80 bg-iron-900/85 backdrop-blur-xl">
        <div className="container-page flex h-16 items-center gap-3">
          <Logo />

          <form onSubmit={submitSearch} className="relative ml-2 hidden flex-1 max-w-md md:block">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-mist-500" />
            <input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="Search forged games…"
              aria-label="Search games"
              className="input pl-9"
            />
          </form>

          <nav className="ml-auto hidden items-center gap-1 md:flex">
            <NavItem to="/" icon={Search} label="Browse" />
            {user && <NavItem to="/library" icon={Library} label="Library" />}
            {user && <NavItem to="/dashboard" icon={LayoutDashboard} label="Studio" />}
            {user && user.role !== 'user' && <NavItem to="/moderation" icon={ShieldAlert} label="Mod" />}
            {user && user.role === 'admin' && <NavItem to="/admin" icon={Sliders} label="Admin" />}
            <Link to="/create" className="btn btn-primary ml-1.5">
              <Plus className="h-4 w-4" />
              Upload game
            </Link>
            {user ? (
              <div className="ml-1.5 flex items-center gap-2">
                <Link
                  to="/dashboard"
                  className="flex h-9 w-9 items-center justify-center rounded-full border border-iron-600 bg-iron-800 text-sm font-700 text-ember-400"
                  title={user.username}
                >
                  {user.username.slice(0, 1).toUpperCase()}
                </Link>
                <button onClick={logout} className="btn btn-ghost px-2.5" aria-label="Sign out">
                  <LogOut className="h-4 w-4" />
                </button>
              </div>
            ) : (
              <Link to="/login" className="btn btn-ghost ml-1.5">
                Sign in
              </Link>
            )}
          </nav>

          <button
            onClick={() => setMobileOpen((v) => !v)}
            className="ml-auto cursor-pointer rounded-lg p-2 text-mist-200 hover:bg-iron-800 md:hidden"
            aria-label="Toggle menu"
          >
            {mobileOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
          </button>
        </div>

        {mobileOpen && (
          <div className="border-t border-iron-700 bg-iron-900 px-4 py-3 md:hidden">
            <form onSubmit={submitSearch} className="relative mb-3">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-mist-500" />
              <input
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder="Search games…"
                aria-label="Search games"
                className="input pl-9"
              />
            </form>
            <div className="flex flex-col gap-1">
              <NavItem to="/" icon={Search} label="Browse" />
              {user && <NavItem to="/library" icon={Library} label="Library" />}
              {user && <NavItem to="/dashboard" icon={LayoutDashboard} label="Studio" />}
              <Link to="/create" onClick={() => setMobileOpen(false)} className="btn btn-primary mt-1">
                <Plus className="h-4 w-4" /> Upload game
              </Link>
              {user ? (
                <button onClick={logout} className="btn btn-ghost mt-1">
                  <LogOut className="h-4 w-4" /> Sign out
                </button>
              ) : (
                <Link to="/login" onClick={() => setMobileOpen(false)} className="btn btn-ghost mt-1">
                  Sign in
                </Link>
              )}
            </div>
          </div>
        )}
      </header>

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="border-t border-iron-700/80 bg-iron-900/60">
        <div className="container-page flex flex-col gap-6 py-10 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <LogoMark className="h-8 w-8" />
            <div>
              <p className="font-display text-sm font-700 text-mist-100">IndieForge</p>
              <p className="text-xs text-mist-500">Forge and play — indie games, browser or download.</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-xs text-mist-500">
            <Hammer className="h-3.5 w-3.5 text-ember-500" />
            Phase 1 prototype · mocked backend · no real charges
          </div>
        </div>
      </footer>
    </div>
  );
}
