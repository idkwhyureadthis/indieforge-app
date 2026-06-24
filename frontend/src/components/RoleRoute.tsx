import { Navigate } from 'react-router-dom';
import type { ReactNode } from 'react';
import type { Role } from '@/lib/types';
import { useAuth } from '@/context/AuthContext';
import { PageLoader } from './ui';

const rank: Record<Role, number> = { user: 0, moderator: 1, admin: 2 };

export function RoleRoute({ min, children }: { min: Role; children: ReactNode }) {
  const { user, loading } = useAuth();
  if (loading) return <PageLoader />;
  if (!user) return <Navigate to="/login" replace />;
  if (rank[user.role] < rank[min]) return <Navigate to="/" replace />;
  return <>{children}</>;
}
