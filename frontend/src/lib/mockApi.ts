// In-browser mock of the future Go/Echo backend.
// State lives in localStorage so created games & purchases survive reloads,
// letting the whole upload→purchase flow be exercised end-to-end.
// Every method mirrors a REST endpoint we'll build in Phase 2.

import type {
  AuthResponse,
  CreateGameInput,
  Game,
  ListFilters,
  Ownership,
  Payment,
  PaymentKind,
  SubscriptionRecord,
  User,
} from './types';
import { SEED_DEV, SEED_GAMES } from './mockData';
import { ApiError } from './errors';
import { bytesToMB, fileToDataURL } from './files';

export { ApiError };

const DB_KEY = 'indieforge_db_v1';

interface StoredUser extends User {
  passwordHash: string;
}
interface RawGame extends Omit<Game, 'stats' | 'owned' | 'subscribed' | 'canPlayFree'> {
  developerId: string;
  developerName: string;
  plays: number;
}
interface DB {
  users: StoredUser[];
  sessions: Record<string, string>; // token -> userId
  games: RawGame[];
  ownerships: Ownership[];
  subscriptions: SubscriptionRecord[];
  payments: Payment[];
}

// ---- latency helper -------------------------------------------------------
const delay = (ms = 320) => new Promise((r) => setTimeout(r, ms));
const uid = (p: string) => `${p}_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 8)}`;

// ---- persistence ----------------------------------------------------------
function seed(): DB {
  const games: RawGame[] = SEED_GAMES.map((g) => ({
    ...g,
    developerId: SEED_DEV.id,
    developerName: SEED_DEV.username,
    plays: 0,
  }));
  const { passwordHash, ...devPublic } = SEED_DEV;
  return {
    users: [{ ...devPublic, passwordHash }],
    sessions: {},
    games,
    ownerships: [],
    subscriptions: [],
    payments: [],
  };
}

function load(): DB {
  try {
    const raw = localStorage.getItem(DB_KEY);
    if (!raw) {
      const fresh = seed();
      localStorage.setItem(DB_KEY, JSON.stringify(fresh));
      return fresh;
    }
    return JSON.parse(raw) as DB;
  } catch {
    return seed();
  }
}

function save(db: DB) {
  localStorage.setItem(DB_KEY, JSON.stringify(db));
}

// ---- helpers --------------------------------------------------------------
function userFromToken(db: DB, token: string | null): StoredUser | null {
  if (!token) return null;
  const id = db.sessions[token];
  return db.users.find((u) => u.id === id) ?? null;
}

function requireUser(db: DB, token: string | null): StoredUser {
  const u = userFromToken(db, token);
  if (!u) throw new ApiError(401, 'Please sign in to continue');
  return u;
}

function publicUser(u: StoredUser): User {
  const { passwordHash, ...rest } = u;
  return rest;
}

function demoActive(g: RawGame): boolean {
  if (!g.demoDay.enabled) return false;
  const t = Date.now();
  const start = g.demoDay.startsAt ? Date.parse(g.demoDay.startsAt) : -Infinity;
  const end = g.demoDay.endsAt ? Date.parse(g.demoDay.endsAt) : Infinity;
  return t >= start && t <= end;
}

function owns(db: DB, userId: string | null, gameId: string): boolean {
  return !!userId && db.ownerships.some((o) => o.userId === userId && o.gameId === gameId);
}
function subscribed(db: DB, userId: string | null, gameId: string): boolean {
  return !!userId && db.subscriptions.some((s) => s.userId === userId && s.gameId === gameId && s.active);
}

function serialize(db: DB, g: RawGame, viewer: StoredUser | null): Game {
  const owners = db.ownerships.filter((o) => o.gameId === g.id).length;
  const subs = db.subscriptions.filter((s) => s.gameId === g.id && s.active).length;
  const active = demoActive(g);
  const owned = owns(db, viewer?.id ?? null, g.id);
  const isSub = subscribed(db, viewer?.id ?? null, g.id);
  return {
    ...g,
    demoDay: { ...g.demoDay, active },
    stats: { owners, subscribers: subs, plays: g.plays ?? 0 },
    owned,
    subscribed: isSub,
    canPlayFree: g.pricingModel === 'free' || owned || isSub || active,
  };
}

function slugify(s: string): string {
  return (
    s
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, '')
      .trim()
      .replace(/\s+/g, '-')
      .slice(0, 60) || 'game'
  );
}

function friendPackPrice(g: RawGame): number {
  return Math.round(g.price * (1 - (g.friendPackDiscount || 0) / 100));
}

// ---------------------------------------------------------------------------
// API surface
// ---------------------------------------------------------------------------
export const mockApi = {
  async register(input: { username: string; email: string; password: string }): Promise<AuthResponse> {
    await delay();
    const db = load();
    const { username, email, password } = input;
    if (!username || !email || !password) throw new ApiError(400, 'Fill in name, email and password');
    if (password.length < 6) throw new ApiError(400, 'Password must be at least 6 characters');
    if (db.users.some((u) => u.email.toLowerCase() === email.toLowerCase()))
      throw new ApiError(409, 'An account with this email already exists');
    if (db.users.some((u) => u.username.toLowerCase() === username.toLowerCase()))
      throw new ApiError(409, 'That username is taken');
    const user: StoredUser = {
      id: uid('usr'),
      username: username.trim(),
      email: email.toLowerCase(),
      role: 'user',
      isDeveloper: false,
      createdAt: new Date().toISOString(),
      passwordHash: password, // mock only — real backend uses bcrypt
    };
    db.users.push(user);
    const token = uid('tok');
    db.sessions[token] = user.id;
    save(db);
    return { token, user: publicUser(user) };
  },

  async login(input: { email: string; password: string }): Promise<AuthResponse> {
    await delay();
    const db = load();
    const user = db.users.find((u) => u.email.toLowerCase() === input.email.toLowerCase());
    if (!user || user.passwordHash !== input.password)
      throw new ApiError(401, 'Invalid email or password');
    const token = uid('tok');
    db.sessions[token] = user.id;
    save(db);
    return { token, user: publicUser(user) };
  },

  async logout(token: string): Promise<void> {
    await delay(120);
    const db = load();
    delete db.sessions[token];
    save(db);
  },

  async me(token: string | null): Promise<User> {
    await delay(120);
    const db = load();
    const user = requireUser(db, token);
    return publicUser(user);
  },

  async listGames(filters: ListFilters = {}, token: string | null = null): Promise<Game[]> {
    await delay();
    const db = load();
    const viewer = userFromToken(db, token);
    let games = db.games.filter((g) => g.status === 'published');

    const q = (filters.search ?? '').trim().toLowerCase();
    if (q)
      games = games.filter(
        (g) =>
          g.title.toLowerCase().includes(q) ||
          g.tagline.toLowerCase().includes(q) ||
          g.developerName.toLowerCase().includes(q) ||
          g.tags.some((t) => t.includes(q)),
      );
    if (filters.genre) games = games.filter((g) => g.genre === filters.genre);
    if (filters.tag) games = games.filter((g) => g.tags.includes(filters.tag!));
    if (filters.pricing === 'free') games = games.filter((g) => g.pricingModel === 'free');
    if (filters.pricing === 'paid') games = games.filter((g) => g.pricingModel === 'paid');
    if (filters.pricing === 'subscription') games = games.filter((g) => g.subscription.enabled);
    if (filters.pricing === 'demo') games = games.filter((g) => demoActive(g));

    const result = games.map((g) => serialize(db, g, viewer));
    switch (filters.sort) {
      case 'popular':
        result.sort((a, b) => b.stats.owners - a.stats.owners);
        break;
      case 'price-asc':
        result.sort((a, b) => a.price - b.price);
        break;
      case 'price-desc':
        result.sort((a, b) => b.price - a.price);
        break;
      default:
        result.sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt));
    }
    return result;
  },

  async getGame(key: string, token: string | null = null): Promise<Game> {
    await delay();
    const db = load();
    const viewer = userFromToken(db, token);
    const g = db.games.find((x) => x.slug === key || x.id === key);
    if (!g) throw new ApiError(404, 'Game not found');
    return serialize(db, g, viewer);
  },

  async createGame(token: string | null, input: CreateGameInput): Promise<Game> {
    await delay(500);
    const db = load();
    const user = requireUser(db, token);
    if (!input.title?.trim()) throw new ApiError(400, 'Give your game a title');
    if (!input.hasBrowserBuild && !input.hasDownloadBuild)
      throw new ApiError(400, 'Add at least one build: browser or downloadable');

    let slug = slugify(input.title);
    let n = 1;
    while (db.games.some((g) => g.slug === slug)) slug = `${slugify(input.title)}-${++n}`;

    // Files → data/object URLs for the mock (real backend uploads to S3).
    const coverImage = input.coverFile ? await fileToDataURL(input.coverFile) : null;
    const screenshots = await Promise.all(input.screenshotFiles.map(fileToDataURL));
    const browserBuildUrl = input.browserBuildFile
      ? URL.createObjectURL(input.browserBuildFile)
      : input.browserBuildUrl;
    const downloadFileName = input.downloadFile?.name ?? null;
    const downloadSizeMB = input.downloadFile ? bytesToMB(input.downloadFile.size) : null;

    const game: RawGame = {
      id: uid('game'),
      slug,
      title: input.title.trim(),
      tagline: input.tagline.trim(),
      description: input.description.trim(),
      genre: input.genre || 'Other',
      tags: input.tags.slice(0, 8),
      developerId: user.id,
      developerName: user.username,
      coverImage,
      screenshots,
      hasBrowserBuild: input.hasBrowserBuild,
      browserBuildUrl,
      hasDownloadBuild: input.hasDownloadBuild,
      downloadFileName,
      downloadSizeMB,
      downloadPlatforms: input.downloadPlatforms,
      supportsMultiplayer: input.supportsMultiplayer,
      pricingModel: input.pricingModel,
      price: input.pricingModel === 'paid' ? Math.max(0, Math.round(input.price)) : 0,
      friendPackDiscount: Math.min(90, Math.max(0, Math.round(input.friendPackDiscount))),
      subscription: {
        enabled: input.subscription.enabled,
        price: input.subscription.enabled ? Math.max(1, Math.round(input.subscription.price)) : 0,
        period: 'month',
        benefits: input.subscription.enabled ? input.subscription.benefits.filter(Boolean).slice(0, 10) : [],
      },
      demoDay: {
        enabled: input.demoDay.enabled,
        startsAt: input.demoDay.enabled ? input.demoDay.startsAt : null,
        endsAt: input.demoDay.enabled ? input.demoDay.endsAt : null,
        active: false,
      },
      theme: input.theme,
      status: 'published',
      createdAt: new Date().toISOString(),
      plays: 0,
    };

    db.games.push(game);
    const idx = db.users.findIndex((u) => u.id === user.id);
    if (idx >= 0) db.users[idx].isDeveloper = true;
    save(db);
    return serialize(db, game, db.users[idx]);
  },

  async myGames(token: string | null): Promise<Game[]> {
    await delay();
    const db = load();
    const user = requireUser(db, token);
    return db.games.filter((g) => g.developerId === user.id).map((g) => serialize(db, g, user));
  },

  async library(token: string | null): Promise<{ owned: Game[]; subscribed: Game[] }> {
    await delay();
    const db = load();
    const user = requireUser(db, token);
    const owned = db.ownerships
      .filter((o) => o.userId === user.id)
      .map((o) => db.games.find((g) => g.id === o.gameId))
      .filter((g): g is RawGame => !!g)
      .map((g) => serialize(db, g, user));
    const subs = db.subscriptions
      .filter((s) => s.userId === user.id && s.active)
      .map((s) => db.games.find((g) => g.id === s.gameId))
      .filter((g): g is RawGame => !!g)
      .map((g) => serialize(db, g, user));
    return { owned, subscribed: subs };
  },

  // Free games & demo-day grant access immediately (no payment).
  async claimFree(token: string | null, gameId: string): Promise<Game> {
    await delay();
    const db = load();
    const user = requireUser(db, token);
    const g = db.games.find((x) => x.id === gameId || x.slug === gameId);
    if (!g) throw new ApiError(404, 'Game not found');
    if (g.pricingModel !== 'free' && !demoActive(g))
      throw new ApiError(400, 'This game is not free');
    if (owns(db, user.id, g.id)) throw new ApiError(409, 'Already in your library');
    db.ownerships.push({
      id: uid('own'),
      userId: user.id,
      gameId: g.id,
      type: demoActive(g) && g.pricingModel === 'paid' ? 'free' : 'free',
      price: 0,
      createdAt: new Date().toISOString(),
    });
    save(db);
    return serialize(db, g, user);
  },

  // ---- Payment flow (mock YooKassa) ----------------------------------------
  // createPayment validates everything, then returns a "pending" payment with
  // a confirmationUrl. The UI redirects there (our mock YooKassa page), which
  // calls confirmPayment — the equivalent of YooKassa's success webhook.
  async createPayment(
    token: string | null,
    input: { gameId: string; kind: PaymentKind; friendUsername?: string },
  ): Promise<Payment> {
    await delay(350);
    const db = load();
    const user = requireUser(db, token);
    const g = db.games.find((x) => x.id === input.gameId || x.slug === input.gameId);
    if (!g) throw new ApiError(404, 'Game not found');

    let amount = 0;
    if (input.kind === 'purchase') {
      if (g.pricingModel !== 'paid') throw new ApiError(400, 'This game is not for sale');
      if (owns(db, user.id, g.id)) throw new ApiError(409, 'Already in your library');
      amount = g.price;
    } else if (input.kind === 'subscription') {
      if (!g.subscription.enabled) throw new ApiError(400, 'This game has no subscription');
      if (subscribed(db, user.id, g.id)) throw new ApiError(409, 'Subscription already active');
      amount = g.subscription.price;
    } else if (input.kind === 'friend-pack') {
      if (!owns(db, user.id, g.id))
        throw new ApiError(403, 'Friend Pack is only available once you own the game');
      const friend = db.users.find(
        (u) => u.username.toLowerCase() === (input.friendUsername ?? '').toLowerCase(),
      );
      if (!friend) throw new ApiError(404, 'No friend found with that username');
      if (friend.id === user.id) throw new ApiError(400, 'You cannot gift a game to yourself');
      if (owns(db, friend.id, g.id)) throw new ApiError(409, 'Your friend already owns this game');
      amount = friendPackPrice(g);
    }

    const payment: Payment = {
      id: uid('pay'),
      gameId: g.id,
      userId: user.id,
      kind: input.kind,
      amount,
      status: 'pending',
      friendUsername: input.friendUsername,
      confirmationUrl: `/checkout/pay/${'PAYMENT_ID'}`,
      createdAt: new Date().toISOString(),
    };
    payment.confirmationUrl = `/checkout/pay/${payment.id}`;
    db.payments.push(payment);
    save(db);
    return payment;
  },

  async getPayment(token: string | null, id: string): Promise<{ payment: Payment; game: Game }> {
    await delay(150);
    const db = load();
    const user = requireUser(db, token);
    const payment = db.payments.find((p) => p.id === id);
    if (!payment || payment.userId !== user.id) throw new ApiError(404, 'Payment not found');
    const g = db.games.find((x) => x.id === payment.gameId)!;
    return { payment, game: serialize(db, g, user) };
  },

  // Mock equivalent of YooKassa's `payment.succeeded` webhook.
  async confirmPayment(token: string | null, id: string): Promise<{ payment: Payment; game: Game }> {
    await delay(700);
    const db = load();
    const user = requireUser(db, token);
    const payment = db.payments.find((p) => p.id === id);
    if (!payment || payment.userId !== user.id) throw new ApiError(404, 'Payment not found');
    if (payment.status === 'succeeded') {
      const g0 = db.games.find((x) => x.id === payment.gameId)!;
      return { payment, game: serialize(db, g0, user) };
    }
    if (payment.status === 'canceled') throw new ApiError(409, 'Payment was canceled');

    const g = db.games.find((x) => x.id === payment.gameId)!;
    const ts = new Date().toISOString();

    if (payment.kind === 'purchase') {
      db.ownerships.push({ id: uid('own'), userId: user.id, gameId: g.id, type: 'purchase', price: payment.amount, createdAt: ts });
    } else if (payment.kind === 'subscription') {
      db.subscriptions.push({ id: uid('sub'), userId: user.id, gameId: g.id, developerId: g.developerId, price: payment.amount, active: true, startedAt: ts });
    } else if (payment.kind === 'friend-pack') {
      const friend = db.users.find((u) => u.username.toLowerCase() === (payment.friendUsername ?? '').toLowerCase());
      if (!friend) throw new ApiError(404, 'Friend no longer exists');
      db.ownerships.push({ id: uid('own'), userId: friend.id, gameId: g.id, type: 'friend-pack', price: payment.amount, giftedBy: user.username, createdAt: ts });
    }
    payment.status = 'succeeded';
    save(db);
    return { payment, game: serialize(db, g, user) };
  },

  async cancelPayment(token: string | null, id: string): Promise<void> {
    await delay(200);
    const db = load();
    const user = requireUser(db, token);
    const payment = db.payments.find((p) => p.id === id);
    if (!payment || payment.userId !== user.id) throw new ApiError(404, 'Payment not found');
    if (payment.status === 'pending') payment.status = 'canceled';
    save(db);
  },

  // Dev helper exposed via the UI: wipe and re-seed.
  async resetDemo(): Promise<void> {
    localStorage.removeItem(DB_KEY);
    load();
  },
};

export type MockApi = typeof mockApi;
