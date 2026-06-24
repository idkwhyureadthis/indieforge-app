import type { Game } from '@/lib/types';

/**
 * Renders a game's cover. When no image was uploaded we synthesize an
 * on-theme gradient "key art" from the game's accent colours, so the catalog
 * never shows a broken image and stays fully offline.
 */
export function CoverArt({
  game,
  className = '',
  showTitle = true,
}: {
  game: Pick<Game, 'title' | 'coverImage' | 'theme' | 'genre'>;
  className?: string;
  showTitle?: boolean;
}) {
  if (game.coverImage) {
    return (
      <img
        src={game.coverImage}
        alt={`${game.title} cover art`}
        loading="lazy"
        className={`h-full w-full object-cover ${className}`}
      />
    );
  }
  const { accent, background } = game.theme;
  const id = game.title.replace(/[^a-z0-9]/gi, '') || 'g';
  return (
    <div
      className={`relative h-full w-full overflow-hidden ${className}`}
      style={{ backgroundColor: background }}
      aria-label={`${game.title} key art`}
    >
      {/* single low-opacity accent wash from one corner — flat, not noisy */}
      <div
        className="absolute inset-0"
        style={{ background: `linear-gradient(135deg, ${accent}22 0%, transparent 55%)` }}
      />
      {/* square forge grid */}
      <svg className="absolute inset-0 h-full w-full opacity-[0.07]" aria-hidden>
        <defs>
          <pattern id={`grid-${id}`} width="24" height="24" patternUnits="userSpaceOnUse">
            <path d="M24 0H0V24" fill="none" stroke="#fff" strokeWidth="1" />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill={`url(#grid-${id})`} />
      </svg>
      {/* small solid accent marker */}
      <span className="absolute left-4 top-4 block h-2.5 w-2.5" style={{ backgroundColor: accent }} />
      {showTitle && (
        <div className="absolute inset-0 flex items-end p-4">
          <span className="font-display text-xl font-700 leading-tight tracking-tight text-white">
            {game.title}
          </span>
        </div>
      )}
    </div>
  );
}
