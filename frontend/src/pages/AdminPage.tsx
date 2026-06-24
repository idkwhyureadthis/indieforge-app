import { useEffect, useState } from 'react';
import { Sliders } from 'lucide-react';
import type { ServiceSettings } from '@/lib/types';
import { api, ApiError } from '@/lib/api';
import { PageLoader, Spinner } from '@/components/ui';
import { useToast } from '@/context/ToastContext';

export function AdminPage() {
  const toast = useToast();
  const [settings, setSettings] = useState<ServiceSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .getSettings()
      .then(setSettings)
      .catch(() => toast('Could not load settings', 'error'))
      .finally(() => setLoading(false));
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

  return (
    <div className="container-page max-w-2xl py-10">
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
            Cut taken from each sale. Snapshotted per payment, so changes don’t affect past transactions.
          </p>
        </div>

        <div className="border-t border-iron-700 pt-5">
          <p className="mb-3 text-sm font-600 text-mist-200">Home page sections</p>
          <Switch
            label="Trending"
            description="Show the trending row (turn on once there’s enough activity)."
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
  );
}

function Switch({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
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
