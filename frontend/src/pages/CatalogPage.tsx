import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Plus, Repeat, Users2, Gift, CalendarClock, Search, X } from 'lucide-react';
import type { Game, HomeSections, ListFilters } from '@/lib/types';
import { api } from '@/lib/api';
import { GameCard } from '@/components/GameCard';
import { CoverArt } from '@/components/CoverArt';
import { EmptyState, PriceTag, SectionTitle, Spinner } from '@/components/ui';
import { GENRES } from '@/lib/constants';

const PRICING_TABS: { key: NonNullable<ListFilters['pricing']>; label: string }[] = [
  { key: '', label: 'All' },
  { key: 'free', label: 'Free' },
  { key: 'paid', label: 'Paid' },
  { key: 'subscription', label: 'Subscription' },
  { key: 'demo', label: 'Demo Day' },
];

const FEATURES = [
  { icon: Repeat, title: 'Author subscriptions', text: 'Back a creator monthly. They set the price and the perks.' },
  { icon: Users2, title: 'Browser multiplayer', text: 'Jump into real-time games with friends — no install.' },
  { icon: Gift, title: 'Friend Pack', text: 'Own a game? Gift a copy to a friend at a discount.' },
  { icon: CalendarClock, title: 'Demo Day', text: 'Free-to-play weekends, straight from the catalog.' },
];

export function CatalogPage() {
  const [params, setParams] = useSearchParams();
  const search = params.get('search') ?? '';
  const [pricing, setPricing] = useState<NonNullable<ListFilters['pricing']>>('');
  const [genre, setGenre] = useState('');
  const [sort, setSort] = useState<NonNullable<ListFilters['sort']>>('new');
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    setLoading(true);
    api.listGames({ search, pricing, genre, sort }).then((g) => {
      if (active) {
        setGames(g);
        setLoading(false);
      }
    });
    return () => {
      active = false;
    };
  }, [search, pricing, genre, sort]);

  const [home, setHome] = useState<HomeSections | null>(null);
  useEffect(() => {
    api.home().then(setHome).catch(() => setHome(null));
  }, []);

  const demoGames = useMemo(() => games.filter((g) => g.demoDay.active), [games]);
  const showHomeRows = !search && pricing === '' && !genre;
  const featured = demoGames[0] ?? games[0];
  const minis = useMemo(() => games.filter((g) => g !== featured).slice(0, 2), [games, featured]);
  const clearSearch = () => setParams({});

  return (
    <div>
      {/* Hero */}
      <section className="relative overflow-hidden border-b border-iron-700/70">
        <div className="container-page grid items-center gap-12 py-12 sm:py-20 lg:grid-cols-2">
          <div className="max-w-2xl">
            <h1 className="text-4xl font-700 leading-[1.05] tracking-tight text-mist-50 sm:text-6xl">
              Forge worlds.
              <br />
              <span className="text-ember-500">Play instantly.</span>
            </h1>
            <p className="mt-5 max-w-xl text-lg text-mist-300">
              A catalog of indie games you can launch right in your browser or download as an app.
              Free, pay-once, or back the creators you love with a subscription.
            </p>
            <div className="mt-7 flex flex-wrap gap-3">
              <a href="#catalog" className="btn btn-primary btn-lg">
                Browse the catalog
              </a>
              <Link to="/create" className="btn btn-ghost btn-lg">
                <Plus className="h-4 w-4" /> Upload your game
              </Link>
            </div>
          </div>

          {/* Featured showcase fills the right side */}
          <div className="hidden lg:block">
            {featured ? (
              <div>
                <Link
                  to={`/game/${featured.slug}`}
                  className="card group block overflow-hidden transition-colors duration-200 hover:border-iron-500"
                >
                  <div className="relative aspect-[16/9]">
                    <CoverArt game={featured} showTitle={false} />
                    <span className="absolute left-3 top-3 rounded-sm bg-iron-950/85 px-2 py-0.5 text-[10px] font-700 uppercase tracking-wide text-ember-400">
                      {featured.demoDay.active ? 'Demo Day' : 'Featured'}
                    </span>
                  </div>
                  <div className="flex items-center justify-between gap-3 p-4">
                    <div className="min-w-0">
                      <h3 className="truncate font-display text-lg font-700 text-mist-50 group-hover:text-ember-400">
                        {featured.title}
                      </h3>
                      <p className="truncate text-sm text-mist-400">{featured.tagline}</p>
                    </div>
                    <PriceTag game={featured} className="shrink-0" />
                  </div>
                </Link>

                {minis.length > 0 && (
                  <div className="mt-4 grid grid-cols-2 gap-4">
                    {minis.map((g) => (
                      <Link
                        key={g.id}
                        to={`/game/${g.slug}`}
                        className="card group overflow-hidden transition-colors duration-200 hover:border-iron-500"
                      >
                        <div className="aspect-[16/10]">
                          <CoverArt game={g} showTitle={false} />
                        </div>
                        <div className="flex items-center justify-between gap-2 p-2.5">
                          <p className="truncate text-sm font-600 text-mist-100">{g.title}</p>
                          <PriceTag game={g} className="shrink-0 text-xs" />
                        </div>
                      </Link>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="card aspect-[4/3] animate-pulse" />
            )}
          </div>
        </div>
      </section>

      {/* Feature strip — compact 2x2 on mobile (titles only), full text from sm up */}
      <section className="border-b border-iron-700/70 bg-iron-900/40">
        <div className="container-page grid grid-cols-2 gap-x-6 gap-y-5 py-7 sm:gap-px sm:gap-y-0 sm:py-0 lg:grid-cols-4">
          {FEATURES.map((f) => (
            <div key={f.title} className="flex flex-col gap-1.5 sm:gap-2 sm:py-8 sm:pr-4">
              <f.icon className="h-5 w-5 text-ember-500 sm:h-6 sm:w-6" />
              <h3 className="text-sm font-700 text-mist-100">{f.title}</h3>
              <p className="hidden text-sm text-mist-400 sm:block">{f.text}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Demo Day spotlight */}
      {demoGames.length > 0 && !search && pricing === '' && (
        <section className="container-page py-10">
          <SectionTitle
            action={
              <span className="inline-flex items-center gap-1.5 text-sm text-emerald-400">
                <CalendarClock className="h-4 w-4" /> Free to play right now
              </span>
            }
          >
            Demo Day
          </SectionTitle>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {demoGames.slice(0, 5).map((g) => (
              <GameCard key={g.id} game={g} />
            ))}
          </div>
        </section>
      )}

      {/* Trending (shown when enabled by admin & non-empty) */}
      {showHomeRows && home && home.trending.length > 0 && (
        <section className="container-page py-10">
          <SectionTitle>Trending</SectionTitle>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {home.trending.slice(0, 5).map((g) => (
              <GameCard key={g.id} game={g} />
            ))}
          </div>
        </section>
      )}

      {/* Most popular */}
      {showHomeRows && home && home.popular.length > 0 && (
        <section className="container-page py-10">
          <SectionTitle>Most popular</SectionTitle>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {home.popular.slice(0, 5).map((g) => (
              <GameCard key={g.id} game={g} />
            ))}
          </div>
        </section>
      )}

      {/* Catalog */}
      <section id="catalog" className="container-page py-10">
        <SectionTitle>{search ? `Results for “${search}”` : 'Browse games'}</SectionTitle>

        {search && (
          <button onClick={clearSearch} className="mb-4 inline-flex items-center gap-1.5 text-sm text-mist-400 hover:text-ember-400">
            <X className="h-4 w-4" /> Clear search
          </button>
        )}

        {/* Filters */}
        <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          {/* pricing: swipeable single row on mobile, wraps on larger screens */}
          <div className="-mx-1 flex gap-1.5 overflow-x-auto px-1 pb-1 [scrollbar-width:none] sm:mx-0 sm:flex-wrap sm:overflow-visible sm:px-0 sm:pb-0">
            {PRICING_TABS.map((t) => (
              <button
                key={t.key || 'all'}
                onClick={() => setPricing(t.key)}
                className={`shrink-0 cursor-pointer whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-500 transition-colors duration-200 ${
                  pricing === t.key
                    ? 'bg-ember-500/15 text-ember-400 ring-1 ring-ember-500/40'
                    : 'text-mist-300 hover:bg-iron-800'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
          {/* genre + sort dropdowns: full-width pair on mobile */}
          <div className="grid grid-cols-2 gap-2 sm:flex">
            <select value={genre} onChange={(e) => setGenre(e.target.value)} className="input w-full py-2 sm:w-auto" aria-label="Filter by genre">
              <option value="">All genres</option>
              {GENRES.map((g) => (
                <option key={g} value={g}>
                  {g}
                </option>
              ))}
            </select>
            <select
              value={sort}
              onChange={(e) => setSort(e.target.value as NonNullable<ListFilters['sort']>)}
              className="input w-full py-2 sm:w-auto"
              aria-label="Sort games"
            >
              <option value="new">Newest</option>
              <option value="popular">Most owned</option>
              <option value="price-asc">Price: low to high</option>
              <option value="price-desc">Price: high to low</option>
            </select>
          </div>
        </div>

        {loading ? (
          <div className="flex justify-center py-16">
            <Spinner className="h-7 w-7 text-ember-500" />
          </div>
        ) : games.length === 0 ? (
          <EmptyState
            icon={<Search className="h-8 w-8" />}
            title="No games match your filters"
            description="Try clearing filters, or be the first to forge something here."
            action={
              <Link to="/create" className="btn btn-primary">
                <Plus className="h-4 w-4" /> Upload a game
              </Link>
            }
          />
        ) : (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {games.map((g) => (
              <GameCard key={g.id} game={g} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
