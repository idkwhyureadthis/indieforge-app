import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ChevronLeft, Gift, Repeat, ShoppingCart, ShieldCheck, Lock } from 'lucide-react';
import type { Game, PaymentKind } from '@/lib/types';
import { api, ApiError } from '@/lib/api';
import { CoverArt } from '@/components/CoverArt';
import { PageLoader, Spinner } from '@/components/ui';
import { RUB } from '@/lib/constants';
import { useToast } from '@/context/ToastContext';

export function CheckoutPage() {
  const { slug } = useParams();
  const [params] = useSearchParams();
  const kind = (params.get('kind') as PaymentKind) || 'purchase';
  const navigate = useNavigate();
  const toast = useToast();

  const [game, setGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);
  const [friend, setFriend] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!slug) return;
    api
      .getGame(slug)
      .then(setGame)
      .finally(() => setLoading(false));
  }, [slug]);

  if (loading) return <PageLoader label="Preparing checkout…" />;
  if (!game) return null;

  const amount =
    kind === 'subscription'
      ? game.subscription.price
      : kind === 'friend-pack'
        ? Math.round(game.price * (1 - game.friendPackDiscount / 100))
        : game.price;

  const meta = {
    purchase: { icon: ShoppingCart, title: 'Buy & keep forever', note: 'One-time payment. Yours to download and replay.' },
    'friend-pack': { icon: Gift, title: 'Friend Pack gift', note: `Discounted copy gifted to a friend (−${game.friendPackDiscount}%).` },
    subscription: { icon: Repeat, title: `Subscribe to ${game.developerName}`, note: 'Recurring monthly support. Cancel anytime.' },
  }[kind];

  const startPayment = async () => {
    setBusy(true);
    try {
      const payment = await api.createPayment({
        gameId: game.id,
        kind,
        friendUsername: kind === 'friend-pack' ? friend.trim() : undefined,
      });
      // Redirect to YooKassa's hosted confirmation page (external URL).
      if (/^https?:\/\//.test(payment.confirmationUrl)) {
        window.location.href = payment.confirmationUrl;
      } else {
        navigate(payment.confirmationUrl);
      }
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not start payment', 'error');
      setBusy(false);
    }
  };

  const Icon = meta.icon;

  return (
    <div className="container-page max-w-3xl py-8">
      <Link to={`/game/${game.slug}`} className="inline-flex items-center gap-1 text-sm text-mist-300 hover:text-mist-50">
        <ChevronLeft className="h-4 w-4" /> Back to {game.title}
      </Link>

      <h1 className="mt-4 text-2xl font-700 text-mist-50">Checkout</h1>

      <div className="mt-6 grid gap-6 md:grid-cols-[1fr_18rem]">
        {/* Order details */}
        <div className="card p-5">
          <div className="flex gap-4">
            <div className="h-20 w-32 shrink-0 overflow-hidden rounded-lg">
              <CoverArt game={game} showTitle={false} />
            </div>
            <div>
              <div className="flex items-center gap-2 text-ember-400">
                <Icon className="h-4 w-4" />
                <span className="text-sm font-600">{meta.title}</span>
              </div>
              <h2 className="mt-1 font-display text-lg font-700 text-mist-50">{game.title}</h2>
              <p className="text-sm text-mist-400">{meta.note}</p>
            </div>
          </div>

          {kind === 'friend-pack' && (
            <div className="mt-5 border-t border-iron-700 pt-4">
              <label htmlFor="friend" className="label">
                Friend’s username
              </label>
              <input
                id="friend"
                value={friend}
                onChange={(e) => setFriend(e.target.value)}
                className="input"
                placeholder="e.g. pixelsmith"
              />
              <p className="mt-1.5 text-xs text-mist-500">
                The game lands directly in their library. They’ll see it was gifted by you.
              </p>
            </div>
          )}

          <div className="mt-5 flex items-start gap-2 rounded-lg border border-iron-700 bg-iron-900/60 p-3 text-xs text-mist-400">
            <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" />
            Payments are processed by YooKassa. In this Phase 1 prototype the payment screen is
            mocked — no card is charged.
          </div>
        </div>

        {/* Summary */}
        <div className="card h-fit p-5">
          <h3 className="text-sm font-700 text-mist-100">Order summary</h3>
          <div className="mt-3 space-y-2 text-sm">
            <div className="flex justify-between text-mist-300">
              <span>{game.title}</span>
              <span className="font-mono">{RUB(kind === 'subscription' ? game.subscription.price : game.price)}</span>
            </div>
            {kind === 'friend-pack' && game.friendPackDiscount > 0 && (
              <div className="flex justify-between text-rose-400">
                <span>Friend Pack discount</span>
                <span className="font-mono">−{game.friendPackDiscount}%</span>
              </div>
            )}
          </div>
          <div className="mt-3 flex items-end justify-between border-t border-iron-700 pt-3">
            <span className="text-sm font-600 text-mist-200">
              {kind === 'subscription' ? 'Per month' : 'Total'}
            </span>
            <span className="font-mono text-xl font-700 text-ember-400">{RUB(amount)}</span>
          </div>

          <button onClick={startPayment} disabled={busy} className="btn btn-primary btn-lg mt-4 w-full">
            {busy ? <Spinner /> : <><Lock className="h-4 w-4" /> Pay with YooKassa</>}
          </button>
          <p className="mt-2 text-center text-[11px] text-mist-500">You’ll confirm on the secure payment page.</p>
        </div>
      </div>
    </div>
  );
}
