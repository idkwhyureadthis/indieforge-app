import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Users2, Repeat, Coins, Gamepad2 } from 'lucide-react';
import type { Game } from '@/lib/types';
import { api } from '@/lib/api';
import { CoverArt } from '@/components/CoverArt';
import { EmptyState, FeatureBadges, PageLoader, SectionTitle } from '@/components/ui';
import { RUB } from '@/lib/constants';
import { useAuth } from '@/context/AuthContext';

export function DashboardPage() {
  const { user } = useAuth();
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .myGames()
      .then(setGames)
      .finally(() => setLoading(false));
  }, []);

  const totals = useMemo(() => {
    return games.reduce(
      (acc, g) => {
        acc.owners += g.stats.owners;
        acc.subs += g.stats.subscribers;
        acc.revenue += g.stats.owners * g.price + g.stats.subscribers * g.subscription.price;
        return acc;
      },
      { owners: 0, subs: 0, revenue: 0 },
    );
  }, [games]);

  if (loading) return <PageLoader label="Loading your studio…" />;

  return (
    <div className="container-page py-10">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-sm text-ember-400">Creator studio</p>
          <h1 className="text-3xl font-700 text-mist-50">{user?.username}</h1>
        </div>
        <Link to="/create" className="btn btn-primary">
          <Plus className="h-4 w-4" /> Upload new game
        </Link>
      </div>

      {/* Stats */}
      <div className="mb-10 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard icon={Gamepad2} label="Games" value={String(games.length)} />
        <StatCard icon={Users2} label="Total owners" value={String(totals.owners)} />
        <StatCard icon={Repeat} label="Subscribers" value={String(totals.subs)} />
        <StatCard icon={Coins} label="Est. revenue" value={RUB(totals.revenue)} accent />
      </div>

      <SectionTitle>Your games</SectionTitle>
      {games.length === 0 ? (
        <EmptyState
          icon={<Gamepad2 className="h-8 w-8" />}
          title="No games published yet"
          description="Forge your first game — upload a browser or downloadable build, set your price, and publish."
          action={
            <Link to="/create" className="btn btn-primary">
              <Plus className="h-4 w-4" /> Upload your first game
            </Link>
          }
        />
      ) : (
        <div className="space-y-3">
          {games.map((g) => (
            <div key={g.id} className="card flex flex-col gap-4 p-4 sm:flex-row sm:items-center">
              <Link to={`/game/${g.slug}`} className="h-20 w-32 shrink-0 overflow-hidden rounded-lg">
                <CoverArt game={g} showTitle={false} />
              </Link>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <Link to={`/game/${g.slug}`} className="font-display text-lg font-600 text-mist-50 hover:text-ember-400">
                    {g.title}
                  </Link>
                  <span className="rounded bg-iron-800 px-2 py-0.5 text-xs text-mist-400">{g.genre}</span>
                </div>
                <p className="line-clamp-1 text-sm text-mist-400">{g.tagline}</p>
                <div className="mt-2">
                  <FeatureBadges game={g} />
                </div>
              </div>
              <div className="flex shrink-0 gap-6 sm:flex-col sm:gap-1 sm:text-right">
                <div>
                  <p className="font-mono text-sm font-700 text-mist-100">{g.stats.owners}</p>
                  <p className="text-[11px] text-mist-500">owners</p>
                </div>
                <div>
                  <p className="font-mono text-sm font-700 text-ember-400">
                    {g.pricingModel === 'free' ? 'Free' : RUB(g.price)}
                  </p>
                  <p className="text-[11px] text-mist-500">price</p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function StatCard({
  icon: Icon,
  label,
  value,
  accent,
}: {
  icon: typeof Users2;
  label: string;
  value: string;
  accent?: boolean;
}) {
  return (
    <div className="card p-4">
      <Icon className={`h-5 w-5 ${accent ? 'text-ember-500' : 'text-mist-400'}`} />
      <p className={`mt-3 font-mono text-2xl font-700 ${accent ? 'text-ember-400' : 'text-mist-50'}`}>{value}</p>
      <p className="text-xs text-mist-500">{label}</p>
    </div>
  );
}
