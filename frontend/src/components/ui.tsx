import type { ReactNode } from 'react';
import { Loader2, Globe, Download, Users2, Repeat, CalendarClock, Gift } from 'lucide-react';
import type { Game } from '@/lib/types';
import { RUB } from '@/lib/constants';

export function Spinner({ className = 'h-5 w-5' }: { className?: string }) {
  return <Loader2 className={`animate-spin ${className}`} aria-hidden />;
}

export function PageLoader({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex min-h-[40vh] flex-col items-center justify-center gap-3 text-mist-400">
      <Spinner className="h-7 w-7 text-ember-500" />
      <p className="text-sm">{label}</p>
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
  icon,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div className="card flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
      {icon && <div className="text-ember-500">{icon}</div>}
      <h3 className="text-lg font-600 text-mist-100">{title}</h3>
      {description && <p className="max-w-md text-sm text-mist-400">{description}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

export function PriceTag({ game, className = '' }: { game: Game; className?: string }) {
  if (game.owned) {
    return <span className={`font-mono text-sm font-600 text-emerald-400 ${className}`}>In library</span>;
  }
  if (game.pricingModel === 'free') {
    return <span className={`font-mono text-sm font-600 text-emerald-400 ${className}`}>Free</span>;
  }
  return <span className={`font-mono text-sm font-600 text-ember-400 ${className}`}>{RUB(game.price)}</span>;
}

/** Small capability badges shown on cards & detail pages. */
export function FeatureBadges({ game, size = 'sm' }: { game: Game; size?: 'sm' | 'md' }) {
  const items: { icon: typeof Globe; label: string; cls: string }[] = [];
  const muted = 'text-mist-300';
  if (game.hasBrowserBuild) items.push({ icon: Globe, label: 'Play in browser', cls: muted });
  if (game.hasDownloadBuild) items.push({ icon: Download, label: 'Download', cls: muted });
  if (game.supportsMultiplayer) items.push({ icon: Users2, label: 'Multiplayer', cls: muted });
  if (game.subscription.enabled) items.push({ icon: Repeat, label: 'Subscription', cls: 'text-ember-400' });
  if (game.demoDay.active) items.push({ icon: CalendarClock, label: 'Demo Day', cls: 'text-ember-400' });
  if (game.friendPackDiscount > 0) items.push({ icon: Gift, label: 'Friend Pack', cls: muted });

  const dim = size === 'md' ? 'h-4 w-4' : 'h-3.5 w-3.5';
  const text = size === 'md' ? 'text-xs' : 'text-[11px]';
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
      {items.map((it) => (
        <span key={it.label} className={`inline-flex items-center gap-1 ${text} font-medium ${it.cls}`}>
          <it.icon className={dim} aria-hidden />
          {it.label}
        </span>
      ))}
    </div>
  );
}

export function Tag({ children }: { children: ReactNode }) {
  return <span className="chip">{children}</span>;
}

export function SectionTitle({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="mb-5 flex items-end justify-between gap-4">
      <h2 className="text-xl font-700 text-mist-50 sm:text-2xl">{children}</h2>
      {action}
    </div>
  );
}
