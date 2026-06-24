import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Play, Download, Gift, Repeat, Library as LibraryIcon } from 'lucide-react';
import type { Game } from '@/lib/types';
import { api } from '@/lib/api';
import { CoverArt } from '@/components/CoverArt';
import { EmptyState, PageLoader, SectionTitle } from '@/components/ui';
import { useToast } from '@/context/ToastContext';

export function LibraryPage() {
  const navigate = useNavigate();
  const toast = useToast();
  const [owned, setOwned] = useState<Game[]>([]);
  const [subs, setSubs] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .library()
      .then(({ owned, subscribed }) => {
        setOwned(owned);
        setSubs(subscribed);
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <PageLoader label="Loading your library…" />;

  return (
    <div className="container-page py-10">
      <h1 className="mb-8 text-3xl font-700 text-mist-50">Your library</h1>

      <SectionTitle>Owned games</SectionTitle>
      {owned.length === 0 ? (
        <EmptyState
          icon={<LibraryIcon className="h-8 w-8" />}
          title="No games yet"
          description="Games you buy or claim for free will show up here, ready to play or download."
          action={
            <Link to="/" className="btn btn-primary">
              Browse the catalog
            </Link>
          }
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {owned.map((g) => (
            <div key={g.id} className="card overflow-hidden">
              <Link to={`/game/${g.slug}`} className="block aspect-[16/9] overflow-hidden">
                <CoverArt game={g} showTitle={false} />
              </Link>
              <div className="p-4">
                <div className="flex items-center justify-between gap-2">
                  <h3 className="font-display font-600 text-mist-50">{g.title}</h3>
                  <span className="text-xs text-mist-500">{g.genre}</span>
                </div>
                <p className="mt-0.5 text-xs text-mist-400">by {g.developerName}</p>
                <div className="mt-3 flex gap-2">
                  {g.hasBrowserBuild && (
                    <button onClick={() => navigate(`/play/${g.slug}`)} className="btn btn-primary flex-1 py-2">
                      <Play className="h-4 w-4" /> Play
                    </button>
                  )}
                  {g.hasDownloadBuild && (
                    <button
                      onClick={async () => {
                        try {
                          window.open(await api.downloadUrl(g.id), '_blank');
                        } catch {
                          toast('Could not start download', 'error');
                        }
                      }}
                      className="btn btn-ghost flex-1 py-2"
                    >
                      <Download className="h-4 w-4" /> Download
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="mt-12">
        <SectionTitle>Subscriptions</SectionTitle>
        {subs.length === 0 ? (
          <EmptyState
            icon={<Repeat className="h-8 w-8" />}
            title="No active subscriptions"
            description="Back your favourite creators monthly to unlock perks they define."
          />
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {subs.map((g) => (
              <Link key={g.id} to={`/game/${g.slug}`} className="card flex items-center gap-3 p-3 hover:border-iron-500">
                <div className="h-16 w-24 shrink-0 overflow-hidden rounded-lg">
                  <CoverArt game={g} showTitle={false} />
                </div>
                <div className="min-w-0">
                  <h3 className="truncate font-600 text-mist-50">{g.title}</h3>
                  <p className="flex items-center gap-1 text-xs text-ember-400">
                    <Repeat className="h-3 w-3" /> Subscribed to {g.developerName}
                  </p>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>

      {owned.some((g) => g.friendPackDiscount > 0) && (
        <div className="mt-12 flex items-center gap-3 rounded-xl border border-rose-500/20 bg-rose-500/5 p-4 text-sm text-rose-300">
          <Gift className="h-5 w-5 shrink-0" />
          Some of your games support the Friend Pack — open a game page to gift a discounted copy.
        </div>
      )}
    </div>
  );
}
