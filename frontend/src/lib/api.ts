// API client — real HTTP calls to the Go/Echo backend.
// The in-browser mock (mockApi.ts) is kept for offline demos but unused here.
import { http } from './http';
import { ApiError } from './errors';
import type {
  AuthResponse,
  CreateGameInput,
  Game,
  HomeSections,
  ListFilters,
  Payment,
  PaymentKind,
  Report,
  ServiceSettings,
  User,
} from './types';

export { ApiError };

const TOKEN_KEY = 'indieforge_token';
let token: string | null = localStorage.getItem(TOKEN_KEY);

export function setToken(t: string | null) {
  token = t;
  if (t) localStorage.setItem(TOKEN_KEY, t);
  else localStorage.removeItem(TOKEN_KEY);
}
export function getToken() {
  return token;
}

function gameQuery(f: ListFilters): string {
  const p = new URLSearchParams();
  if (f.search) p.set('search', f.search);
  if (f.genre) p.set('genre', f.genre);
  if (f.tag) p.set('tag', f.tag);
  if (f.pricing) p.set('pricing', f.pricing);
  if (f.sort) p.set('sort', f.sort);
  const s = p.toString();
  return s ? `?${s}` : '';
}

function buildGameForm(input: CreateGameInput): FormData {
  const f = new FormData();
  f.set('title', input.title);
  f.set('tagline', input.tagline);
  f.set('description', input.description);
  f.set('genre', input.genre);
  f.set('tags', JSON.stringify(input.tags));
  f.set('hasBrowserBuild', String(input.hasBrowserBuild));
  if (input.browserBuildUrl) f.set('browserBuildUrl', input.browserBuildUrl);
  f.set('hasDownloadBuild', String(input.hasDownloadBuild));
  f.set('downloadPlatforms', JSON.stringify(input.downloadPlatforms));
  f.set('supportsMultiplayer', String(input.supportsMultiplayer));
  f.set('pricingModel', input.pricingModel);
  f.set('price', String(input.price));
  f.set('friendPackDiscount', String(input.friendPackDiscount));
  f.set('subEnabled', String(input.subscription.enabled));
  f.set('subPrice', String(input.subscription.price));
  f.set('subBenefits', JSON.stringify(input.subscription.benefits));
  if (input.subscription.chatLink) f.set('subChatLink', input.subscription.chatLink);
  f.set('demoEnabled', String(input.demoDay.enabled));
  if (input.demoDay.startsAt) f.set('demoStartsAt', input.demoDay.startsAt);
  if (input.demoDay.endsAt) f.set('demoEndsAt', input.demoDay.endsAt);
  f.set('theme', JSON.stringify(input.theme));
  if (input.coverFile) f.set('cover', input.coverFile);
  input.screenshotFiles.forEach((s) => f.append('screenshots', s));
  if (input.browserBuildFile) f.set('browserBuild', input.browserBuildFile);
  if (input.downloadFile) f.set('downloadFile', input.downloadFile);
  return f;
}

export const api = {
  // ---- auth ----
  register: (input: { username: string; email: string; password: string }) =>
    http.post<AuthResponse>('/auth/register', input),
  login: (input: { email: string; password: string }) => http.post<AuthResponse>('/auth/login', input),
  logout: () => http.post<void>('/auth/logout'),
  me: () => http.get<{ user: User }>('/auth/me').then((r) => r.user),

  // ---- catalog ----
  home: () => http.get<HomeSections>('/home'),
  listGames: (filters: ListFilters = {}) =>
    http.get<{ games: Game[] }>(`/games${gameQuery(filters)}`).then((r) => r.games),
  getGame: (key: string) => http.get<{ game: Game }>(`/games/${key}`).then((r) => r.game),
  createGame: (input: CreateGameInput) =>
    http.postForm<{ game: Game }>('/games', buildGameForm(input)).then((r) => r.game),
  myGames: () => http.get<{ games: Game[] }>('/me/games').then((r) => r.games),
  downloadUrl: (gameId: string) => http.get<{ url: string }>(`/games/${gameId}/download`).then((r) => r.url),

  // ---- commerce ----
  library: () => http.get<{ owned: Game[]; subscribed: Game[] }>('/me/library'),
  claimFree: (gameId: string) => http.post<{ game: Game }>(`/games/${gameId}/claim-free`).then((r) => r.game),
  createPayment: (input: { gameId: string; kind: PaymentKind; friendUsername?: string }) =>
    http.post<Payment>('/payments', input),
  getPayment: (id: string) => http.get<{ payment: Payment; game: Game }>(`/payments/${id}`),
  cancelPayment: (id: string) => http.post<void>(`/payments/${id}/cancel`),
  perks: (gameId: string) => http.get<{ chatLink: string }>(`/games/${gameId}/perks`).then((r) => r.chatLink),

  // ---- moderation ----
  createReport: (input: { targetType: string; targetId: string; reason: string; details: string }) =>
    http.post<{ report: Report }>('/reports', input).then((r) => r.report),
  listReports: (status = '') =>
    http.get<{ reports: Report[] }>(`/moderation/reports${status ? `?status=${status}` : ''}`).then((r) => r.reports),
  getReport: (id: string) => http.get<{ report: Report }>(`/moderation/reports/${id}`).then((r) => r.report),
  resolveReport: (id: string, action: string, note: string) =>
    http.post<{ report: Report }>(`/moderation/reports/${id}/resolve`, { action, note }).then((r) => r.report),

  // ---- admin settings ----
  getSettings: () => http.get<{ settings: ServiceSettings }>('/admin/settings').then((r) => r.settings),
  updateSettings: (s: ServiceSettings) =>
    http.put<{ settings: ServiceSettings }>('/admin/settings', s).then((r) => r.settings),
};
