import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, Users2, Maximize2 } from 'lucide-react';
import type { Game } from '@/lib/types';
import { api } from '@/lib/api';
import { PageLoader } from '@/components/ui';

export function PlayPage() {
  const { slug } = useParams();
  const [game, setGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!slug) return;
    api
      .getGame(slug)
      .then(setGame)
      .finally(() => setLoading(false));
  }, [slug]);

  if (loading) return <PageLoader label="Spinning up the game…" />;
  if (!game) return null;

  const allowed = game.canPlayFree;

  return (
    <div className="bg-iron-950">
      <div className="container-page py-4">
        <div className="mb-3 flex items-center justify-between">
          <Link to={`/game/${game.slug}`} className="inline-flex items-center gap-1.5 text-sm text-mist-300 hover:text-mist-50">
            <ArrowLeft className="h-4 w-4" /> {game.title}
          </Link>
          {game.supportsMultiplayer && (
            <span className="inline-flex items-center gap-1.5 rounded-lg bg-sky-500/15 px-2.5 py-1 text-xs font-600 text-sky-300">
              <Users2 className="h-3.5 w-3.5" /> Browser multiplayer enabled
            </span>
          )}
        </div>

        {!allowed ? (
          <div className="card flex aspect-video flex-col items-center justify-center gap-3 text-center">
            <p className="text-mist-200">You need access to play this game.</p>
            <Link to={`/checkout/${game.slug}`} className="btn btn-primary">
              Get access
            </Link>
          </div>
        ) : game.browserBuildUrl ? (
          <iframe
            title={`${game.title} — play`}
            src={game.browserBuildUrl}
            className="aspect-video w-full rounded-2xl border border-iron-700 bg-black"
            allow="autoplay; fullscreen; gamepad"
          />
        ) : (
          <DemoCanvas game={game} />
        )}
      </div>
    </div>
  );
}

/** Stand-in for a real browser build (none uploaded in the seed data). */
function DemoCanvas({ game }: { game: Game }) {
  return (
    <div
      className="relative flex aspect-video w-full items-center justify-center overflow-hidden rounded-2xl border border-iron-700"
      style={{ background: `radial-gradient(120% 120% at 50% 0%, ${game.theme.accent}33, ${game.theme.background})` }}
    >
      <div className="absolute right-3 top-3">
        <button className="btn btn-ghost px-2.5 py-2" aria-label="Fullscreen">
          <Maximize2 className="h-4 w-4" />
        </button>
      </div>
      <div className="text-center">
        <div
          className="mx-auto mb-4 h-16 w-16 animate-pulse rounded-2xl"
          style={{ background: `linear-gradient(135deg, ${game.theme.accent}, ${game.theme.accent2})` }}
        />
        <p className="font-display text-xl font-700 text-white">{game.title}</p>
        <p className="mt-1 text-sm text-white/60">
          Browser build runs here. (Real builds load from S3 in Phase 2.)
        </p>
      </div>
    </div>
  );
}
