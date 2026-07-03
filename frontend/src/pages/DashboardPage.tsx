import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Users2, Repeat, Coins, Gamepad2, Wallet, ArrowDownToLine, Clock, CheckCircle2, XCircle, Key, Trash2 } from 'lucide-react';
import type { APIKey, Game, Payout, PayoutBalance } from '@/lib/types';
import { api } from '@/lib/api';
import { CoverArt } from '@/components/CoverArt';
import { EmptyState, FeatureBadges, PageLoader, SectionTitle } from '@/components/ui';
import { RUB, RUBAmount } from '@/lib/constants';
import { useAuth } from '@/context/AuthContext';
import { useToast } from '@/context/ToastContext';

export function DashboardPage() {
  const { user } = useAuth();
  const showToast = useToast();
  const [games, setGames] = useState<Game[]>([]);
  const [balance, setBalance] = useState<PayoutBalance | null>(null);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [payoutAmount, setPayoutAmount] = useState('');
  const [requesting, setRequesting] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [creatingKey, setCreatingKey] = useState(false);
  const [revealedKey, setRevealedKey] = useState<{ id: string; value: string } | null>(null);

  useEffect(() => {
    const promises: Promise<any>[] = [api.myGames()];
    if (user?.isDeveloper) {
      promises.push(api.getPayoutBalance(), api.listAPIKeys());
    } else {
      promises.push(Promise.resolve(null), Promise.resolve([]));
    }
    Promise.all(promises)
      .then(([g, b, k]) => { setGames(g); setBalance(b); setApiKeys(k ?? []); })
      .finally(() => setLoading(false));
  }, [user]);

  const handleCreateKey = async () => {
    if (!newKeyName.trim()) return;
    setCreatingKey(true);
    try {
      const res = await api.createAPIKey(newKeyName.trim());
      setRevealedKey({ id: res.id, value: res.key });
      setNewKeyName('');
      const keys = await api.listAPIKeys();
      setApiKeys(keys);
      showToast('API key created — copy it now, it won\'t be shown again', 'success');
    } catch (e: any) {
      showToast(e.message || 'Failed to create key', 'error');
    } finally {
      setCreatingKey(false);
    }
  };

  const handleRevokeKey = async (id: string) => {
    if (!confirm('Revoke this API key? Games using it will lose access immediately.')) return;
    try {
      await api.revokeAPIKey(id);
      setApiKeys(prev => prev.filter(k => k.id !== id));
      if (revealedKey?.id === id) setRevealedKey(null);
      showToast('Key revoked', 'success');
    } catch {
      showToast('Failed to revoke key', 'error');
    }
  };

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

  const handleRequestPayout = async () => {
    const amount = Math.round(parseFloat(payoutAmount));
    if (!amount || amount <= 0) return;
    setRequesting(true);
    try {
      await api.requestPayout(amount);
      showToast('Payout request submitted', 'success');
      setPayoutAmount('');
      const b = await api.getPayoutBalance();
      setBalance(b);
    } catch (e: any) {
      showToast(e.message || 'Failed to request payout', 'error');
    } finally {
      setRequesting(false);
    }
  };

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
        <StatCard icon={Coins} label="Est. revenue" value={RUBAmount(totals.revenue)} accent />
      </div>

      {/* Payouts */}
      {user?.isDeveloper && balance !== null && (
        <div className="mb-10">
          <SectionTitle>Payouts</SectionTitle>
          <div className="grid gap-4 lg:grid-cols-3">
            {/* Balance card */}
            <div className="card col-span-1 p-5">
              <div className="flex items-center gap-2 text-mist-400">
                <Wallet className="h-4 w-4" />
                <span className="text-xs font-600 uppercase tracking-wider">Balance</span>
              </div>
              <p className="mt-3 font-mono text-3xl font-700 text-ember-400">{RUBAmount(balance.available)}</p>
              <p className="mt-1 text-xs text-mist-500">
                Total earned: {RUBAmount(balance.earned)} · Requested: {RUBAmount(balance.earned - balance.available)}
              </p>
              <div className="mt-4 flex gap-2">
                <input
                  type="number"
                  min="1"
                  step="0.01"
                  placeholder="Amount, ₽"
                  value={payoutAmount}
                  onChange={e => setPayoutAmount(e.target.value)}
                  className="input min-w-0 flex-1 text-sm"
                  disabled={balance.available <= 0}
                />
                <button
                  onClick={handleRequestPayout}
                  disabled={requesting || balance.available <= 0 || !payoutAmount}
                  className="btn btn-primary shrink-0"
                >
                  <ArrowDownToLine className="h-4 w-4" />
                  {requesting ? 'Sending…' : 'Withdraw'}
                </button>
              </div>
              {balance.available <= 0 && (
                <p className="mt-2 text-xs text-mist-500">No available balance to withdraw.</p>
              )}
            </div>

            {/* History */}
            <div className="card col-span-2 p-5">
              <p className="mb-3 text-xs font-600 uppercase tracking-wider text-mist-400">Payout history</p>
              {balance.history.length === 0 ? (
                <p className="text-sm text-mist-500">No payout requests yet.</p>
              ) : (
                <div className="space-y-2">
                  {balance.history.map(p => (
                    <PayoutRow key={p.ID} payout={p} />
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Developer API keys */}
      {user?.isDeveloper && (
        <div className="mb-10">
          <SectionTitle>Developer API keys</SectionTitle>
          <p className="mb-4 text-sm text-mist-400">
            Use these keys to verify player subscriptions from your game's backend.{' '}
            <Link to="/docs#developer-api" className="text-ember-400 hover:underline">API docs →</Link>
          </p>

          {/* Create new key */}
          <div className="mb-4 flex gap-2">
            <input
              type="text"
              placeholder="Key name, e.g. Production"
              value={newKeyName}
              onChange={e => setNewKeyName(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleCreateKey()}
              className="input min-w-0 flex-1 text-sm"
            />
            <button
              onClick={handleCreateKey}
              disabled={creatingKey || !newKeyName.trim()}
              className="btn btn-primary shrink-0"
            >
              <Key className="h-4 w-4" />
              {creatingKey ? 'Creating…' : 'Create key'}
            </button>
          </div>

          {/* Newly revealed key */}
          {revealedKey && (
            <div className="mb-4 rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-4">
              <p className="mb-1 text-xs font-600 text-emerald-400">Copy this key — it will never be shown again:</p>
              <div className="flex items-center gap-2">
                <code className="flex-1 break-all rounded bg-iron-900 px-3 py-2 text-sm text-emerald-300">
                  {revealedKey.value}
                </code>
                <button
                  onClick={() => { navigator.clipboard.writeText(revealedKey.value); showToast('Copied!', 'success'); }}
                  className="btn btn-ghost shrink-0 text-xs"
                >
                  Copy
                </button>
              </div>
            </div>
          )}

          {/* Keys list */}
          {apiKeys.length === 0 ? (
            <p className="text-sm text-mist-500">No API keys yet.</p>
          ) : (
            <div className="space-y-2">
              {apiKeys.map(k => (
                <div key={k.id} className="flex items-center gap-3 rounded-xl bg-iron-800/40 px-4 py-3">
                  <Key className="h-4 w-4 shrink-0 text-mist-500" />
                  <div className="min-w-0 flex-1">
                    <span className="font-600 text-mist-100">{k.name}</span>
                    <span className="ml-2 font-mono text-xs text-mist-500">sk_••••••••</span>
                  </div>
                  <span className="text-xs text-mist-500">
                    {k.lastUsedAt
                      ? `Last used ${new Date(k.lastUsedAt).toLocaleDateString()}`
                      : `Created ${new Date(k.createdAt).toLocaleDateString()}`}
                  </span>
                  <button
                    onClick={() => handleRevokeKey(k.id)}
                    title="Revoke key"
                    className="rounded-lg p-1.5 text-mist-500 hover:bg-red-500/10 hover:text-red-400"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Games list */}
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

function PayoutRow({ payout }: { payout: Payout }) {
  const statusIcon = {
    pending: <Clock className="h-3.5 w-3.5 text-yellow-400" />,
    paid:    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />,
    rejected:<XCircle className="h-3.5 w-3.5 text-red-400" />,
  }[payout.Status];

  const statusColor = {
    pending: 'text-yellow-400',
    paid:    'text-emerald-400',
    rejected:'text-red-400',
  }[payout.Status];

  return (
    <div className="flex items-center justify-between rounded-lg bg-iron-800/40 px-4 py-2.5 text-sm">
      <div className="flex items-center gap-2">
        {statusIcon}
        <span className={`font-600 ${statusColor}`}>{payout.Status}</span>
        {payout.Note && <span className="text-mist-500">— {payout.Note}</span>}
      </div>
      <div className="flex items-center gap-4 text-right">
        <span className="font-mono font-700 text-mist-100">{RUB(payout.Amount)}</span>
        <span className="text-xs text-mist-500">{new Date(payout.CreatedAt).toLocaleDateString()}</span>
      </div>
    </div>
  );
}

function StatCard({
  icon: Icon, label, value, accent,
}: { icon: typeof Users2; label: string; value: string; accent?: boolean }) {
  return (
    <div className="card p-4">
      <Icon className={`h-5 w-5 ${accent ? 'text-ember-500' : 'text-mist-400'}`} />
      <p className={`mt-3 font-mono text-2xl font-700 ${accent ? 'text-ember-400' : 'text-mist-50'}`}>{value}</p>
      <p className="text-xs text-mist-500">{label}</p>
    </div>
  );
}
