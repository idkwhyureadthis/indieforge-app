import { Link } from 'react-router-dom';
import { Users2, Download, Globe } from 'lucide-react';
import type { Game } from '@/lib/types';
import { CoverArt } from './CoverArt';
import { PriceTag } from './ui';

export function GameCard({ game }: { game: Game }) {
  return (
    <Link
      to={`/game/${game.slug}`}
      className="group card cursor-pointer overflow-hidden transition-all duration-200 hover:-translate-y-0.5 hover:border-iron-500 hover:shadow-[0_18px_40px_-20px_rgba(0,0,0,0.8)]"
    >
      <div className="relative aspect-[16/10] overflow-hidden">
        <CoverArt game={game} showTitle={false} className="transition-transform duration-300 group-hover:scale-[1.04]" />
        {/* corner ribbons */}
        <div className="absolute left-2 top-2 flex flex-wrap gap-1.5">
          {game.demoDay.active && (
            <span className="rounded-sm bg-iron-950/85 px-2 py-0.5 text-[10px] font-700 uppercase tracking-wide text-ember-400">
              Demo Day
            </span>
          )}
          {game.subscription.enabled && (
            <span className="rounded-sm bg-iron-950/85 px-2 py-0.5 text-[10px] font-700 uppercase tracking-wide text-mist-200">
              Sub
            </span>
          )}
        </div>
        <div className="absolute bottom-2 right-2 flex items-center gap-1.5">
          {game.hasBrowserBuild && <Globe className="h-4 w-4 text-white/85 drop-shadow" aria-label="Playable in browser" />}
          {game.hasDownloadBuild && <Download className="h-4 w-4 text-white/85 drop-shadow" aria-label="Downloadable" />}
          {game.supportsMultiplayer && <Users2 className="h-4 w-4 text-white/85 drop-shadow" aria-label="Multiplayer" />}
        </div>
      </div>

      <div className="flex flex-col gap-1.5 p-3.5">
        <div className="flex items-start justify-between gap-2">
          <h3 className="line-clamp-1 font-display text-base font-600 text-mist-50 transition-colors group-hover:text-ember-400">
            {game.title}
          </h3>
          <PriceTag game={game} className="shrink-0" />
        </div>
        <p className="line-clamp-2 text-xs leading-relaxed text-mist-400">{game.tagline}</p>
        <div className="mt-1 flex items-center justify-between text-[11px] text-mist-500">
          <span className="truncate">by {game.developerName}</span>
          <span className="rounded bg-iron-800 px-1.5 py-0.5 text-mist-400">{game.genre}</span>
        </div>
      </div>
    </Link>
  );
}
