import { Link } from 'react-router-dom';

/** IndieForge emblem — cropped from the brand artwork (served from /public). */
export function LogoMark({ className = 'h-9 w-9' }: { className?: string }) {
  return <img src="/logo-mark.png" alt="IndieForge" className={`object-contain ${className}`} />;
}

export function Logo() {
  return (
    <Link to="/" className="group flex items-center gap-2.5">
      <LogoMark className="h-9 w-9 text-ember-500" />
      <span className="flex flex-col leading-none">
        <span className="font-display text-lg font-700 tracking-tight text-mist-50">
          Indie<span className="text-ember-500">Forge</span>
        </span>
        <span className="text-[10px] font-medium uppercase tracking-[0.2em] text-mist-500">
          forge and play
        </span>
      </span>
    </Link>
  );
}
