import { Navigate, useLocation } from 'react-router-dom';
import type { ReactNode } from 'react';
import { useAuth } from '@/context/AuthContext';
import { PageLoader } from './ui';

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth();
  const location = useLocation();
  if (loading) return <PageLoader />;
  if (!user) return <Navigate to="/login" state={{ from: location.pathname + location.search }} replace />;
  return <>{children}</>;
}
