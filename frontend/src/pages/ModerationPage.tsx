import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { ShieldAlert, EyeOff, Trash2, Check } from 'lucide-react';
import type { Report } from '@/lib/types';
import { api, ApiError } from '@/lib/api';
import { EmptyState, PageLoader, SectionTitle, Spinner } from '@/components/ui';
import { useToast } from '@/context/ToastContext';

const STATUSES = ['open', 'resolved', 'dismissed', ''] as const;
const LABELS: Record<string, string> = { open: 'Open', resolved: 'Resolved', dismissed: 'Dismissed', '': 'All' };

export function ModerationPage() {
  const toast = useToast();
  const [status, setStatus] = useState<(typeof STATUSES)[number]>('open');
  const [reports, setReports] = useState<Report[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    api
      .listReports(status)
      .then(setReports)
      .catch(() => toast('Could not load reports', 'error'))
      .finally(() => setLoading(false));
  }, [status, toast]);

  useEffect(() => {
    load();
  }, [load]);

  const resolve = async (id: string, action: string, note: string) => {
    setBusy(id);
    try {
      await api.resolveReport(id, action, note);
      toast('Report resolved', 'success');
      load();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not resolve', 'error');
    } finally {
      setBusy(null);
    }
  };

  return (
    <div className="container-page py-10">
      <div className="mb-6 flex items-center gap-2">
        <ShieldAlert className="h-6 w-6 text-ember-500" />
        <h1 className="text-3xl font-700 text-mist-50">Moderation</h1>
      </div>

      <div className="mb-6 flex flex-wrap gap-1.5">
        {STATUSES.map((s) => (
          <button
            key={s || 'all'}
            onClick={() => setStatus(s)}
            className={`cursor-pointer rounded-md px-3 py-1.5 text-sm font-500 transition-colors ${
              status === s ? 'bg-ember-500/15 text-ember-400 ring-1 ring-ember-500/40' : 'text-mist-300 hover:bg-iron-800'
            }`}
          >
            {LABELS[s]}
          </button>
        ))}
      </div>

      {loading ? (
        <PageLoader label="Loading reports…" />
      ) : reports.length === 0 ? (
        <EmptyState icon={<Check className="h-8 w-8" />} title="Nothing to review" description="No reports match this filter." />
      ) : (
        <div className="space-y-3">
          {reports.map((r) => (
            <div key={r.id} className="card p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="rounded bg-rose-500/15 px-2 py-0.5 text-xs font-700 uppercase text-rose-400">{r.reason}</span>
                    <span className="rounded bg-iron-800 px-2 py-0.5 text-xs text-mist-400">{r.status}</span>
                  </div>
                  <p className="mt-2 text-sm text-mist-200">{r.details || <span className="text-mist-500">No details provided.</span>}</p>
                  <p className="mt-1 text-xs text-mist-500">
                    Target:{' '}
                    <Link to={`/game/${r.targetId}`} className="font-mono text-ember-400 hover:underline">
                      {r.targetId}
                    </Link>{' '}
                    · {new Date(r.createdAt).toLocaleString()}
                  </p>
                </div>
                {r.status === 'open' && (
                  <div className="flex shrink-0 gap-2">
                    <button onClick={() => resolve(r.id, 'dismiss', '')} disabled={busy === r.id} className="btn btn-ghost py-2">
                      {busy === r.id ? <Spinner className="h-4 w-4" /> : 'Dismiss'}
                    </button>
                    <button onClick={() => resolve(r.id, 'hide-game', 'Hidden after report')} disabled={busy === r.id} className="btn btn-ghost py-2 text-amber-400">
                      <EyeOff className="h-4 w-4" /> Hide
                    </button>
                    <button onClick={() => resolve(r.id, 'remove-game', 'Removed after report')} disabled={busy === r.id} className="btn btn-ghost py-2 text-rose-400">
                      <Trash2 className="h-4 w-4" /> Remove
                    </button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="mt-8">
        <SectionTitle>Moderator tools</SectionTitle>
        <p className="text-sm text-mist-400">Hiding a game removes it from the catalog; removing takes it down entirely.</p>
      </div>
    </div>
  );
}
