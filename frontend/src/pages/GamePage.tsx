import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  Globe, Download, Play, Users2, Gift, Repeat, CalendarClock, Check, ChevronLeft, Flag, MessageCircle,
} from 'lucide-react';
import type { Game } from '@/lib/types';
import { api, ApiError } from '@/lib/api';
import { CoverArt } from '@/components/CoverArt';
import { FeatureBadges, PageLoader, Tag } from '@/components/ui';
import { RUB } from '@/lib/constants';
import { useAuth } from '@/context/AuthContext';
import { useToast } from '@/context/ToastContext';

export function GamePage() {
  const { slug } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const toast = useToast();
  const [game, setGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [activeShot, setActiveShot] = useState(0);
  const [reporting, setReporting] = useState(false);

  const reload = useCallback(async () => {
    if (!slug) return;
    const g = await api.getGame(slug);
    setGame(g);
  }, [slug]);

  useEffect(() => {
    setLoading(true);
    reload().finally(() => setLoading(false));
  }, [reload]);

  if (loading) return <PageLoader label="Loading game…" />;
  if (!game) return null;

  const requireAuth = (next: string) => {
    if (!user) {
      navigate('/login', { state: { from: next } });
      return false;
    }
    return true;
  };

  const claimFree = async () => {
    if (!requireAuth(`/game/${game.slug}`)) return;
    setBusy(true);
    try {
      await api.claimFree(game.id);
      toast('Added to your library', 'success');
      await reload();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not add game', 'error');
    } finally {
      setBusy(false);
    }
  };

  const download = async () => {
    if (!requireAuth(`/game/${game.slug}`)) return;
    try {
      const url = await api.downloadUrl(game.id);
      window.open(url, '_blank');
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not start download', 'error');
    }
  };

  const openChat = async () => {
    try {
      const link = await api.perks(game.id);
      window.open(link, '_blank');
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not open chat', 'error');
    }
  };

  const submitReport = async (reason: string, details: string) => {
    try {
      await api.createReport({ targetType: 'game', targetId: game.id, reason, details });
      toast('Report submitted — thanks for keeping IndieForge safe', 'success');
      setReporting(false);
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not submit report', 'error');
    }
  };

  const accent = game.theme.accent;
  const accentStyle = { backgroundColor: game.theme.accent, color: '#0b0b0f' };

  return (
    <div style={{ backgroundColor: game.theme.background }}>
      {/* Banner — flat tint, no heavy blur */}
      <div className="relative">
        <div className="absolute inset-0 overflow-hidden">
          <div className="h-full w-full opacity-20">
            <CoverArt game={game} showTitle={false} />
          </div>
          <div className="absolute inset-0" style={{ background: `linear-gradient(180deg, transparent, ${game.theme.background})` }} />
        </div>

        <div className="container-page relative pt-6">
          <Link to="/" className="inline-flex items-center gap-1 text-sm text-white/70 hover:text-white">
            <ChevronLeft className="h-4 w-4" /> Catalog
          </Link>

          <div className="grid gap-8 py-8 lg:grid-cols-[1fr_22rem]">
            {/* Left: media + info */}
            <div>
              <div className="overflow-hidden rounded-2xl border border-white/10 shadow-2xl">
                <div className="aspect-[16/9]">
                  {game.screenshots.length > 0 ? (
                    <img src={game.screenshots[activeShot]} alt={`${game.title} screenshot ${activeShot + 1}`} className="h-full w-full object-cover" />
                  ) : (
                    <CoverArt game={game} showTitle={false} />
                  )}
                </div>
              </div>

              {game.screenshots.length > 1 && (
                <div className="mt-3 flex gap-2 overflow-x-auto pb-1">
                  {game.screenshots.map((s, i) => (
                    <button
                      key={i}
                      onClick={() => setActiveShot(i)}
                      className={`h-16 w-28 shrink-0 cursor-pointer overflow-hidden rounded-lg border-2 transition-colors ${
                        i === activeShot ? 'border-ember-500' : 'border-transparent opacity-70 hover:opacity-100'
                      }`}
                    >
                      <img src={s} alt={`Thumbnail ${i + 1}`} className="h-full w-full object-cover" />
                    </button>
                  ))}
                </div>
              )}

              <div className="mt-8">
                <h1 className="text-3xl font-700 tracking-tight text-white sm:text-4xl">{game.title}</h1>
                <p className="mt-2 text-lg text-white/70">{game.tagline}</p>
                <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-white/60">
                  <span>
                    by <span className="font-600 text-white/90">{game.developerName}</span>
                  </span>
                  <span className="rounded bg-white/10 px-2 py-0.5 text-white/80">{game.genre}</span>
                  <span>{game.stats.owners} owners</span>
                </div>
                <div className="mt-4">
                  <FeatureBadges game={game} size="md" />
                </div>
              </div>

              <div className="prose-invert mt-8 max-w-none whitespace-pre-line text-[15px] leading-relaxed text-mist-200">
                {game.description || 'No description provided.'}
              </div>

              {game.tags.length > 0 && (
                <div className="mt-6 flex flex-wrap gap-2">
                  {game.tags.map((t) => (
                    <Tag key={t}>#{t}</Tag>
                  ))}
                </div>
              )}

              <button
                onClick={() => (user ? setReporting(true) : navigate('/login', { state: { from: `/game/${game.slug}` } }))}
                className="mt-6 inline-flex items-center gap-1.5 text-xs text-mist-500 transition-colors hover:text-rose-400"
              >
                <Flag className="h-3.5 w-3.5" /> Report this game
              </button>
            </div>

            {/* Right: purchase / play panel */}
            <aside className="lg:sticky lg:top-20 lg:self-start">
              <div className="card overflow-hidden">
                <div className="hidden aspect-[16/10] lg:block">
                  <CoverArt game={game} showTitle={false} />
                </div>
                <div className="space-y-3 p-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-mist-400">
                      {game.pricingModel === 'free' ? 'Price' : 'Buy & keep forever'}
                    </span>
                    <span className="font-mono text-lg font-700" style={{ color: accent }}>
                      {game.pricingModel === 'free' ? 'Free' : RUB(game.price)}
                    </span>
                  </div>

                  {game.owned ? (
                    <div className="space-y-2">
                      <div className="flex items-center gap-2 rounded-lg bg-emerald-500/10 px-3 py-2 text-sm font-600 text-emerald-400">
                        <Check className="h-4 w-4" /> In your library
                      </div>
                      {game.hasBrowserBuild && (
                        <button onClick={() => navigate(`/play/${game.slug}`)} className="btn w-full" style={accentStyle}>
                          <Play className="h-4 w-4" /> Play in browser
                        </button>
                      )}
                      {game.hasDownloadBuild && (
                        <button onClick={download} className="btn btn-ghost w-full">
                          <Download className="h-4 w-4" /> Download {game.downloadSizeMB ? `(${game.downloadSizeMB} MB)` : ''}
                        </button>
                      )}
                    </div>
                  ) : game.pricingModel === 'free' ? (
                    <button onClick={claimFree} disabled={busy} className="btn w-full" style={accentStyle}>
                      <Download className="h-4 w-4" /> Add to library
                    </button>
                  ) : (
                    <div className="space-y-2">
                      {game.demoDay.active && (
                        <button onClick={() => navigate(`/play/${game.slug}`)} className="btn btn-violet w-full">
                          <CalendarClock className="h-4 w-4" /> Play free — Demo Day
                        </button>
                      )}
                      <button onClick={() => navigate(`/checkout/${game.slug}`)} className="btn w-full" style={accentStyle}>
                        Buy for {RUB(game.price)}
                      </button>
                    </div>
                  )}

                  {/* Play buttons for free, unowned games too */}
                  {!game.owned && game.pricingModel === 'free' && game.hasBrowserBuild && (
                    <button onClick={() => navigate(`/play/${game.slug}`)} className="btn btn-ghost w-full">
                      <Globe className="h-4 w-4" /> Play now
                    </button>
                  )}
                </div>
              </div>

              {/* Friend Pack */}
              {game.friendPackDiscount > 0 && game.pricingModel === 'paid' && (
                <div className="card mt-4 p-4">
                  <div className="mb-1 flex items-center gap-2">
                    <Gift className="h-4 w-4 text-rose-400" />
                    <h3 className="text-sm font-700 text-mist-100">Friend Pack</h3>
                    <span className="ml-auto rounded bg-rose-500/15 px-2 py-0.5 text-xs font-700 text-rose-400">
                      −{game.friendPackDiscount}%
                    </span>
                  </div>
                  <p className="text-xs text-mist-400">
                    {game.owned
                      ? 'Gift a copy to a friend at a discount.'
                      : 'Once you own this game, gift a discounted copy to a friend.'}
                  </p>
                  {game.owned && (
                    <button
                      onClick={() => navigate(`/checkout/${game.slug}?kind=friend-pack`)}
                      className="btn btn-ghost mt-3 w-full"
                    >
                      Gift for {RUB(Math.round(game.price * (1 - game.friendPackDiscount / 100)))}
                    </button>
                  )}
                </div>
              )}

              {/* Subscription */}
              {game.subscription.enabled && (
                <div className="card mt-4 p-4">
                  <div className="mb-2 flex items-center gap-2">
                    <Repeat className="h-4 w-4 text-ember-400" />
                    <h3 className="text-sm font-700 text-mist-100">Support {game.developerName}</h3>
                  </div>
                  <p className="font-mono text-2xl font-700 text-ember-400">
                    {RUB(game.subscription.price)}
                    <span className="text-sm font-500 text-mist-500">/{game.subscription.period}</span>
                  </p>
                  <ul className="mt-3 space-y-1.5">
                    {game.subscription.benefits.map((b) => (
                      <li key={b} className="flex items-start gap-2 text-sm text-mist-300">
                        <Check className="mt-0.5 h-4 w-4 shrink-0 text-ember-500" /> {b}
                      </li>
                    ))}
                  </ul>
                  {game.subscribed ? (
                    <div className="mt-3 space-y-2">
                      <div className="flex items-center gap-2 rounded-lg bg-emerald-500/10 px-3 py-2 text-sm font-600 text-emerald-400">
                        <Check className="h-4 w-4" /> Subscription active
                      </div>
                      <button onClick={openChat} className="btn btn-ghost w-full">
                        <MessageCircle className="h-4 w-4" /> Chat with {game.developerName}
                      </button>
                    </div>
                  ) : (
                    <button
                      onClick={() => navigate(`/checkout/${game.slug}?kind=subscription`)}
                      className="btn btn-primary mt-3 w-full"
                    >
                      Subscribe
                    </button>
                  )}
                </div>
              )}

              {game.supportsMultiplayer && (
                <div className="mt-4 flex items-center gap-2 rounded-xl border border-sky-500/20 bg-sky-500/5 px-3 py-2.5 text-sm text-sky-300">
                  <Users2 className="h-4 w-4 shrink-0" /> Real-time multiplayer, playable in your browser.
                </div>
              )}
            </aside>
          </div>
        </div>
      </div>

      {reporting && <ReportModal onClose={() => setReporting(false)} onSubmit={submitReport} />}
    </div>
  );
}

const REASONS: { value: string; label: string }[] = [
  { value: 'inappropriate', label: 'Inappropriate content' },
  { value: 'copyright', label: 'Copyright infringement' },
  { value: 'broken', label: 'Broken / does not work' },
  { value: 'scam', label: 'Scam or malware' },
  { value: 'other', label: 'Other' },
];

function ReportModal({ onClose, onSubmit }: { onClose: () => void; onSubmit: (reason: string, details: string) => void }) {
  const [reason, setReason] = useState('inappropriate');
  const [details, setDetails] = useState('');
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-iron-950/70 p-4" onClick={onClose}>
      <div className="card w-full max-w-md p-6" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-lg font-700 text-mist-50">Report this game</h2>
        <p className="mt-1 text-sm text-mist-400">Tell our moderators what’s wrong.</p>
        <div className="mt-4 space-y-3">
          <select value={reason} onChange={(e) => setReason(e.target.value)} className="input" aria-label="Reason">
            {REASONS.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </select>
          <textarea
            value={details}
            onChange={(e) => setDetails(e.target.value)}
            className="input min-h-24 resize-y"
            placeholder="Details (optional)"
          />
        </div>
        <div className="mt-5 flex gap-2">
          <button onClick={onClose} className="btn btn-ghost flex-1">
            Cancel
          </button>
          <button onClick={() => onSubmit(reason, details)} className="btn btn-primary flex-1">
            Submit report
          </button>
        </div>
      </div>
    </div>
  );
}
