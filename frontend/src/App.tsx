import { Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ProtectedRoute } from './components/ProtectedRoute';
import { RoleRoute } from './components/RoleRoute';
import { CatalogPage } from './pages/CatalogPage';
import { GamePage } from './pages/GamePage';
import { CreateGamePage } from './pages/CreateGamePage';
import { CheckoutPage } from './pages/CheckoutPage';
import { ReturnPage } from './pages/ReturnPage';
import { LibraryPage } from './pages/LibraryPage';
import { DashboardPage } from './pages/DashboardPage';
import { PlayPage } from './pages/PlayPage';
import { AuthPage } from './pages/AuthPage';
import { ModerationPage } from './pages/ModerationPage';
import { AdminPage } from './pages/AdminPage';
import { NotFoundPage } from './pages/NotFoundPage';

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<CatalogPage />} />
        <Route path="/game/:slug" element={<GamePage />} />
        <Route path="/play/:slug" element={<PlayPage />} />
        <Route path="/login" element={<AuthPage mode="login" />} />
        <Route path="/register" element={<AuthPage mode="register" />} />
        <Route
          path="/create"
          element={
            <ProtectedRoute>
              <CreateGamePage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/checkout/:slug"
          element={
            <ProtectedRoute>
              <CheckoutPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/checkout/return"
          element={
            <ProtectedRoute>
              <ReturnPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/library"
          element={
            <ProtectedRoute>
              <LibraryPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/moderation"
          element={
            <RoleRoute min="moderator">
              <ModerationPage />
            </RoleRoute>
          }
        />
        <Route
          path="/admin"
          element={
            <RoleRoute min="admin">
              <AdminPage />
            </RoleRoute>
          }
        />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
