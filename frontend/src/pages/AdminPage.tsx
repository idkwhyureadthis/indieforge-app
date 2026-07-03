import { useEffect, useState } from 'react';
import { Sliders, Wallet, CheckCircle2, XCircle } from 'lucide-react';
import type { PayoutWithDev, ServiceSettings } from '@/lib/types';
import { api, ApiError } from '@/lib/api';
import { PageLoader, SectionTitle, Spinner } from '@/components/ui';
import { useToast } from '@/context/ToastContext';
import { RUB } from '@/lib/constants';

export function AdminPage() {
  const toast = useToast();
  const [settings, setSettings] = useState<ServiceSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [payouts, setPayouts] = useState<PayoutWithDev[]>([]);
  const [payoutsLoading, setPayoutsLoading] = useState(true);
  const [actionId, setActionId] = useState<string | null>(null);
  const [noteMap, setNoteMap] = useState<Record<string, string>>({});

  useEffect(() => {
    api
      .getSettings()
      .then(setSettings)
      .catch(() => toast('Could not load settings', 'error'))
      .finally(() => setLoading(false));

    api
      .adminListPayouts()
      .then(setPayouts)
      .catch(() => toast('Could not load payouts', 'error'))
      .finally(() => setPayoutsLoading(false));
  }, [toast]);

  if (loading) return <PageLoader label="Loading settings…" />;
  if (!settings) return null;

  const set = (patch: Partial<ServiceSettings>) => setSettings({ ...settings, ...patch });

  const save = async () => {
    setSaving(true);
    try {
      const updated = await api.updateSettings(settings);
      setSettings(updated);
      toast('Settings saved', 'success');
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not save', 'error');
    } finally {
      setSaving(false);
    }
  };

  const handlePayout = async (id: string, status: 'paid' | 'rejected') => {
    setActionId(id);
    try {
      const updated = await api.adminUpdatePayout(id, status, noteMap[id] || '');
      setPayouts(prev => prev.map(p => p.ID === id ? { ...p, ...updated } : p));
      toast(`Payout marked as ${status}`, 'success');
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Action failed', 'error');
    } finally {
      setActionId(null);
    }
  };

  const pending = payouts.filter(p => p.Status === 'pending');
  const resolved = payouts.filter(p => p.Status !== 'pending');

  return (
    <div className="container-page max-w-3xl py-10 space-y-10">
      {/* Settings */}
      <div>
        <div className="mb-6 flex items-center gap-2">
          <Sliders className="h-6 w-6 text-ember-500" />
          <h1 className="text-3xl font-700 text-mist-50">Service settings</h1>
        </div>

        <div className="card space-y-6 p-6">
          <div>
            <label htmlFor="commission" className="label">
              Service commission (%)
            </label>
            <input
              id="commission"
              type="number"
              min={0}
              max={100}
              value={settings.commissionPercent}
              onChange={(e) => set({ commissionPercent: Number(e.target.value) })}
              className="input w-40 font-mono"
            />
            <p className="mt-1.5 text-xs text-mist-500">
              Cut taken from each sale. Snapshotted per payment, so changes don't affect past transactions.
            </p>
          </div>

          <div className="border-t border-iron-700 pt-5">
            <p className="mb-3 text-sm font-600 text-mist-200">Home page sections</p>
            <Switch
              label="Trending"
              description="Show the trending row (turn on once there's enough activity)."
              checked={settings.trendingEnabled}
              onChange={(v) => set({ trendingEnabled: v })}
            />
            <Switch
              label="Most popular"
              description="Show the most-popular row."
              checked={settings.popularEnabled}
              onChange={(v) => set({ popularEnabled: v })}
            />
          </div>

          <button onClick={save} disabled={saving} className="btn btn-primary btn-lg w-full">
            {saving ? <Spinner /> : 'Save settings'}
          </button>
        </div>
      </div>

      {/* Payouts */}
      <div>
        <div className="mb-6 flex items-center gap-2">
          <Wallet className="h-6 w-6 text-ember-500" />
          <h1 className="text-2xl font-700 text-mist-50">Payout requests</h1>
        </div>

        {payoutsLoading ? (
          <div className="flex justify-center py-8"><Spinner /></div>
        ) : payouts.length === 0 ? (
          <p className="text-mist-500 text-sm">No payout requests yet.</p>
        ) : (
          <div className="space-y-8">
            {pending.length > 0 && (
              <div>
                <SectionTitle>Pending ({pending.length})</SectionTitle>
                <div className="space-y-3">
                  {pending.map(p => (
                    <div key={p.ID} className="card p-4 space-y-3">
                      <div className="flex items-center justify-between">
                        <div>
                          <span className="font-600 text-mist-100">{p.DeveloperUsername}</span>
                          <span className="ml-3 font-mono text-lg font-700 text-ember-400">{RUB(p.Amount)}</span>
                        </div>
                        <span className="text-xs text-mist-500">{new Date(p.CreatedAt).toLocaleDateString()}</span>
                      </div>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          placeholder="Note (optional)"
                          value={noteMap[p.ID] || ''}
                          onChange={e => setNoteMap(prev => ({ ...prev, [p.ID]: e.target.value }))}
                          className="input flex-1 text-sm"
                        />
                        <button
                          onClick={() => handlePayout(p.ID, 'paid')}
                          disabled={actionId === p.ID}
                          className="btn btn-primary shrink-0"
                        >
                          <CheckCircle2 className="h-4 w-4" />
                          Mark paid
                        </button>
                        <button
                          onClick={() => handlePayout(p.ID, 'rejected')}
                          disabled={actionId === p.ID}
                          className="btn btn-ghost shrink-0 text-red-400 hover:bg-red-500/10"
                        >
                          <XCircle className="h-4 w-4" />
                          Reject
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {resolved.length > 0 && (
              <div>
                <SectionTitle>Resolved</SectionTitle>
                <div className="space-y-2">
                  {resolved.map(p => {
                    const isPaid = p.Status === 'paid';
                    return (
                      <div key={p.ID} className="flex items-center justify-between rounded-xl border border-iron-700/60 bg-iron-800/30 px-4 py-3 text-sm">
                        <div className="flex items-center gap-2">
                          {isPaid
                            ? <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
                            : <XCircle className="h-3.5 w-3.5 text-red-400" />}
                          <span className="font-600 text-mist-200">{p.DeveloperUsername}</span>
                          {p.Note && <span className="text-mist-500">— {p.Note}</span>}
                        </div>
                        <div className="flex items-center gap-4">
                          <span className="font-mono font-700 text-mist-100">{RUB(p.Amount)}</span>
                          <span className={`text-xs font-600 ${isPaid ? 'text-emerald-400' : 'text-red-400'}`}>{p.Status}</span>
                          <span className="text-xs text-mist-500">{new Date(p.CreatedAt).toLocaleDateString()}</span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function Switch({
  label, description, checked, onChange,
}: { label: string; description: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      className="mb-2 flex w-full cursor-pointer items-center gap-3 rounded-xl border border-iron-700 bg-iron-900/40 p-3.5 text-left transition-colors hover:border-iron-600"
    >
      <div className="min-w-0 flex-1">
        <p className="text-sm font-600 text-mist-100">{label}</p>
        <p className="text-xs text-mist-400">{description}</p>
      </div>
      <span className={`relative h-6 w-11 shrink-0 rounded-full transition-colors ${checked ? 'bg-ember-500' : 'bg-iron-600'}`}>
        <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${checked ? 'translate-x-5' : 'translate-x-0.5'}`} />
      </span>
    </button>
  );
}
