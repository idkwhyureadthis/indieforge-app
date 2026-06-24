import { Link } from 'react-router-dom';
import { LogoMark } from '@/components/Logo';

export function NotFoundPage() {
  return (
    <div className="container-page flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
      <LogoMark className="h-14 w-14 opacity-70" />
      <h1 className="text-3xl font-700 text-mist-50">Lost in the forge</h1>
      <p className="max-w-sm text-mist-400">This page didn’t survive the anvil. Let’s get you back to the catalog.</p>
      <Link to="/" className="btn btn-primary btn-lg">
        Back to catalog
      </Link>
    </div>
  );
}
