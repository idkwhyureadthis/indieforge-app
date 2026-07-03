import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  Globe, Download, Play, Users2, Gift, Repeat, CalendarClock, Check, ChevronLeft, Flag, MessageCircle, Maximize2, Key, Copy, X,
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
  const [launchToken, setLaunchToken] = useState<string | null>(null);
  const [generatingToken, setGeneratingToken] = useState(false);

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

  const getLaunchToken = async () => {
    if (!requireAuth(`/game/${game.slug}`)) return;
    setGeneratingToken(true);
    try {
      const { token } = await api.issueLaunchToken(game.id);
      setLaunchToken(token);
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not generate token', 'error');
    } finally {
      setGeneratingToken(false);
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
  const accentStyle = { backgroundColor: accent, color: '#0b0b0f' };
  const wallpaper = game.theme.backgroundImage || game.coverImage;
  const showEmbed = game.hasBrowserBuild && game.canPlayFree;

  return (
    <>
      {/* Full-page wallpaper layer */}
      <div
        className="fixed inset-0 -z-10"
        style={{
          backgroundColor: game.theme.background,
          ...(wallpaper && {
            backgroundImage: `url(${wallpaper})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center top',
            backgroundAttachment: 'fixed',
          }),
        }}
      />

      {/* Center mat */}
      <div
        className="relative mx-auto min-h-screen w-full max-w-5xl shadow-2xl"
        style={{
          backgroundColor: game.theme.background,
          boxShadow: wallpaper ? `0 0 120px 40px ${game.theme.background}` : undefined,
        }}
      >
        {/* Back link */}
        <div className="px-6 pt-5">
          <Link
            to="/"
            className="inline-flex items-center gap-1 text-sm text-white/60 transition-colors hover:text-white"
          >
            <ChevronLeft className="h-4 w-4" /> Catalog
          </Link>
        </div>

        {/* ── Game embed / cover ───────────────────────────────────── */}
        {showEmbed ? (
          <div className="relative mt-4">
            {game.browserBuildUrl ? (
              <iframe
                title={`${game.title} — play`}
                src={game.browserBuildUrl}
                className="aspect-video w-full bg-black"
                allow="autoplay; fullscreen; gamepad"
              />
            ) : (
              <EmbedPlaceholder game={game} onPlay={() => navigate(`/play/${game.slug}`)} accent={accent} />
            )}
            <a
              href={`/play/${game.slug}`}
              target="_blank"
              rel="noopener noreferrer"
              className="absolute bottom-3 right-3 flex items-center gap-1.5 rounded-lg bg-black/60 px-3 py-1.5 text-xs text-white/80 backdrop-blur-sm transition-colors hover:bg-black/80 hover:text-white"
            >
              <Maximize2 className="h-3.5 w-3.5" /> Full screen
            </a>
          </div>
        ) : (
          <div className="relative mt-4 aspect-video w-full overflow-hidden">
            <CoverArt game={game} showTitle={false} />
            {!game.owned && game.pricingModel === 'paid' && (
              <div className="absolute inset-0 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                <button
                  onClick={() => navigate(`/checkout/${game.slug}`)}
                  className="btn px-8 py-3 text-base font-700"
                  style={accentStyle}
                >
                  Buy for {RUB(game.price)}
                </button>
              </div>
            )}
          </div>
        )}

        {/* ── Main content + sidebar ───────────────────────────────── */}
        <div className="grid gap-8 px-6 py-8 lg:grid-cols-[1fr_22rem]">
          {/* Left: title + description */}
          <div>
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

            <div className="prose-invert mt-8 max-w-none whitespace-pre-line text-[15px] leading-relaxed text-mist-200">
              {game.description || 'No description provided.'}
            </div>

            {game.tags.length > 0 && (
              <div className="mt-6 flex flex-wrap gap-2">
                {game.tags.map((t) => (
                  <Tag key={t}>{t}</Tag>
                ))}
              </div>
            )}

            <button
              onClick={() =>
                user ? setReporting(true) : navigate('/login', { state: { from: `/game/${game.slug}` } })
              }
              className="mt-6 inline-flex items-center gap-1.5 text-xs text-mist-500 transition-colors hover:text-rose-400"
            >
              <Flag className="h-3.5 w-3.5" /> Report this game
            </button>
          </div>

          {/* Right: purchase / play panel */}
          <aside className="lg:sticky lg:top-20 lg:self-start">
            <PurchasePanel
              game={game}
              busy={busy}
              accentStyle={accentStyle}
              onClaimFree={claimFree}
              onDownload={download}
              onOpenChat={openChat}
              onGetLaunchToken={getLaunchToken}
              generatingToken={generatingToken}
              navigate={navigate}
            />
            {launchToken && (
              <LaunchTokenModal token={launchToken} onClose={() => setLaunchToken(null)} />
            )}
          </aside>
        </div>

        {/* ── Screenshots ─────────────────────────────────────────── */}
        {game.screenshots.length > 0 && (
          <div className="border-t border-white/5 px-6 pb-10 pt-6">
            <h2 className="mb-4 text-sm font-600 uppercase tracking-widest text-mist-500">Screenshots</h2>

            {/* Active screenshot */}
            <div className="overflow-hidden rounded-xl border border-white/10 shadow-xl">
              <img
                src={game.screenshots[activeShot]}
                alt={`Screenshot ${activeShot + 1}`}
                className="aspect-video w-full object-cover"
              />
            </div>

            {/* Thumbnails */}
            {game.screenshots.length > 1 && (
              <div className="mt-3 flex gap-2 overflow-x-auto">
                {game.screenshots.map((src, i) => (
                  <button
                    key={i}
                    onClick={() => setActiveShot(i)}
                    className={`h-16 w-24 shrink-0 overflow-hidden rounded-lg border-2 transition-colors ${
                      i === activeShot ? 'border-white/50' : 'border-transparent opacity-60 hover:opacity-100'
                    }`}
                  >
                    <img src={src} alt="" className="h-full w-full object-cover" />
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {reporting && <ReportModal onClose={() => setReporting(false)} onSubmit={submitReport} />}
    </>
  );
}

/* ── Purchase panel ────────────────────────────────────────────────────────── */
function PurchasePanel({
  game, busy, accentStyle, onClaimFree, onDownload, onOpenChat, onGetLaunchToken, generatingToken, navigate,
}: {
  game: Game;
  busy: boolean;
  accentStyle: React.CSSProperties;
  onClaimFree: () => void;
  onDownload: () => void;
  onOpenChat: () => void;
  onGetLaunchToken: () => void;
  generatingToken: boolean;
  navigate: ReturnType<typeof useNavigate>;
}) {
  const accent = game.theme.accent;

  return (
    <>
      <div className="card overflow-hidden">
        {/* Cover thumbnail in sidebar */}
        <div className="aspect-[16/10] overflow-hidden">
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
                <div className="space-y-1.5">
                  <button onClick={onDownload} className="btn btn-ghost w-full">
                    <Download className="h-4 w-4" /> Download{' '}
                    {game.downloadSizeMB ? `(${game.downloadSizeMB} MB)` : ''}
                  </button>
                  <button
                    onClick={onGetLaunchToken}
                    disabled={generatingToken}
                    title="Generate a one-time token so the game can identify you"
                    className="flex w-full items-center justify-center gap-2 rounded-lg px-3 py-1.5 text-xs text-mist-400 transition-colors hover:bg-iron-700 hover:text-mist-200 disabled:opacity-50"
                  >
                    <Key className="h-3.5 w-3.5" />
                    {generatingToken ? 'Generating…' : 'Get launch token'}
                  </button>
                </div>
              )}
            </div>
          ) : game.pricingModel === 'free' ? (
            <div className="space-y-2">
              <button onClick={onClaimFree} disabled={busy} className="btn w-full" style={accentStyle}>
                <Download className="h-4 w-4" /> Add to library
              </button>
              {game.hasBrowserBuild && (
                <button onClick={() => navigate(`/play/${game.slug}`)} className="btn btn-ghost w-full">
                  <Globe className="h-4 w-4" /> Play now
                </button>
              )}
            </div>
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
        </div>
      </div>

      {/* Subscription card */}
      {game.subscription.enabled && (
        <div className="card mt-4 p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm text-mist-300">
              <Repeat className="h-4 w-4 text-ember-400" />
              <span>Monthly subscription</span>
            </div>
            <span className="font-mono font-700" style={{ color: accent }}>
              {RUB(game.subscription.price)}<span className="text-xs text-mist-400">/mo</span>
            </span>
          </div>
          {game.subscription.benefits.length > 0 && (
            <ul className="mt-2 space-y-1">
              {game.subscription.benefits.map((b) => (
                <li key={b} className="flex items-center gap-1.5 text-xs text-mist-400">
                  <Check className="h-3 w-3 text-ember-400 shrink-0" />{b}
                </li>
              ))}
            </ul>
          )}
          {game.subscribed ? (
            <p className="mt-3 text-center text-xs text-emerald-400">✓ Subscribed</p>
          ) : (
            <button
              onClick={() => navigate(`/checkout/${game.slug}?kind=subscription`)}
              className="btn btn-ghost mt-3 w-full text-sm"
            >
              <Repeat className="h-4 w-4" /> Subscribe
            </button>
          )}
        </div>
      )}

      {/* Friend Pack */}
      {game.friendPackDiscount > 0 && (
        <div className="card mt-4 p-4">
          <div className="flex items-center gap-2 text-sm">
            <Gift className="h-4 w-4 text-rose-400" />
            <span className="text-mist-300">Friend Pack — {game.friendPackDiscount}% off</span>
          </div>
          <button
            onClick={() => navigate(`/checkout/${game.slug}?kind=friend-pack`)}
            className="btn btn-ghost mt-3 w-full text-sm"
          >
            Gift to a friend
          </button>
        </div>
      )}

      {/* Perks chat link */}
      {game.subscribed && (
        <button onClick={onOpenChat} className="card mt-4 flex w-full items-center gap-2 p-3 text-sm hover:border-iron-500">
          <MessageCircle className="h-4 w-4 text-sky-400" />
          <span className="text-mist-300">Open subscriber chat</span>
        </button>
      )}

      {/* Stats */}
      <div className="mt-4 flex gap-3">
        <div className="card flex-1 p-3 text-center">
          <Users2 className="mx-auto h-4 w-4 text-mist-500" />
          <p className="mt-1 font-mono text-sm font-700 text-mist-100">{game.stats.owners}</p>
          <p className="text-[11px] text-mist-500">owners</p>
        </div>
        <div className="card flex-1 p-3 text-center">
          <Repeat className="mx-auto h-4 w-4 text-mist-500" />
          <p className="mt-1 font-mono text-sm font-700 text-mist-100">{game.stats.subscribers}</p>
          <p className="text-[11px] text-mist-500">subs</p>
        </div>
        <div className="card flex-1 p-3 text-center">
          <Play className="mx-auto h-4 w-4 text-mist-500" />
          <p className="mt-1 font-mono text-sm font-700 text-mist-100">{game.stats.plays}</p>
          <p className="text-[11px] text-mist-500">plays</p>
        </div>
      </div>
    </>
  );
}

/* ── Launch Token Modal ────────────────────────────────────────────────────── */
function LaunchTokenModal({ token, onClose }: { token: string; onClose: () => void }) {
  const toast = useToast();
  const copy = () => {
    navigator.clipboard.writeText(token);
    toast('Token copied!', 'success');
  };
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm">
      <div className="card w-full max-w-md p-6">
        <div className="mb-4 flex items-start justify-between gap-3">
          <div>
            <h3 className="font-display text-lg font-700 text-mist-50">Launch token</h3>
            <p className="mt-1 text-sm text-mist-400">
              Valid for <strong className="text-mist-200">15 minutes</strong>. One-time use — paste into the game when prompted.
            </p>
          </div>
          <button onClick={onClose} className="shrink-0 rounded-lg p-1 text-mist-500 hover:text-mist-200">
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex gap-2 rounded-xl border border-iron-600 bg-iron-900 p-3">
          <code className="flex-1 break-all text-sm text-emerald-300">{token}</code>
          <button
            onClick={copy}
            className="shrink-0 self-start rounded-lg p-1.5 text-mist-400 hover:bg-iron-700 hover:text-mist-100"
          >
            <Copy className="h-4 w-4" />
          </button>
        </div>
        <p className="mt-3 text-xs text-mist-500">
          Never share this token — it temporarily identifies you to the game server.
        </p>
      </div>
    </div>
  );
}

/* ── Embed placeholder ─────────────────────────────────────────────────────── */
function EmbedPlaceholder({ game, onPlay, accent }: { game: Game; onPlay: () => void; accent: string }) {
  return (
    <div
      className="relative flex aspect-video w-full cursor-pointer items-center justify-center overflow-hidden bg-black/60"
      onClick={onPlay}
    >
      <CoverArt game={game} showTitle={false} />
      <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-black/50 backdrop-blur-sm">
        <button
          className="flex h-16 w-16 items-center justify-center rounded-full shadow-xl transition-transform hover:scale-110"
          style={{ backgroundColor: accent }}
        >
          <Play className="h-7 w-7 text-black" />
        </button>
        <p className="absolute bottom-4 text-sm text-white/50">Click to launch</p>
      </div>
    </div>
  );
}

/* ── Report Modal ──────────────────────────────────────────────────────────── */
function ReportModal({
  onClose,
  onSubmit,
}: {
  onClose: () => void;
  onSubmit: (reason: string, details: string) => void;
}) {
  const [reason, setReason] = useState('');
  const [details, setDetails] = useState('');
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/60 p-4 backdrop-blur-sm sm:items-center">
      <div className="card w-full max-w-md p-6">
        <h3 className="mb-4 font-display text-lg font-700 text-mist-50">Report this game</h3>
        <select
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          className="input mb-3 w-full"
        >
          <option value="">Select a reason…</option>
          {['Malware / virus', 'Copyright infringement', 'Inappropriate content', 'Spam', 'Other'].map((r) => (
            <option key={r} value={r}>{r}</option>
          ))}
        </select>
        <textarea
          value={details}
          onChange={(e) => setDetails(e.target.value)}
          placeholder="Additional details (optional)"
          rows={3}
          className="input mb-4 w-full resize-none"
        />
        <div className="flex gap-2">
          <button onClick={onClose} className="btn btn-ghost flex-1">Cancel</button>
          <button
            onClick={() => reason && onSubmit(reason, details)}
            disabled={!reason}
            className="btn btn-primary flex-1"
          >
            Submit report
          </button>
        </div>
      </div>
    </div>
  );
}
