// Domain types — mirror the planned Go/sqlc backend so the mock API can later
// be swapped for real fetch() calls with no component changes.

export type PricingModel = 'free' | 'paid';

export type Role = 'user' | 'moderator' | 'admin';

export interface User {
  id: string;
  username: string;
  email: string;
  role: Role;
  isDeveloper: boolean;
  createdAt: string;
}

export interface SubscriptionPlan {
  enabled: boolean;
  price: number; // RUB per month, set by the author
  period: string;
  benefits: string[];
  chatLink?: string; // perk — only present on create input, never in public game DTO
}

export interface DemoDay {
  enabled: boolean;
  startsAt: string | null;
  endsAt: string | null;
  active: boolean; // computed by the API for the current time
}

/** itch.io-style per-game page customisation. */
export interface GameTheme {
  accent: string; // hex — drives the cover gradient + page accents
  accent2: string; // hex — second gradient stop
  background: string; // hex — center mat / panel background
  backgroundImage?: string; // URL — wallpaper shown on sides behind the mat
  layout: 'classic' | 'immersive';
  cardShape: 'sharp' | 'rounded';
}

export interface Game {
  id: string;
  slug: string;
  title: string;
  tagline: string;
  description: string;
  genre: string;
  tags: string[];
  developerId: string;
  developerName: string;
  coverImage: string | null;
  screenshots: string[];
  hasBrowserBuild: boolean;
  browserBuildUrl: string | null; // object URL or hosted URL of the HTML build
  hasDownloadBuild: boolean;
  downloadFileName: string | null;
  downloadSizeMB: number | null;
  downloadPlatforms: string[];
  supportsMultiplayer: boolean;
  pricingModel: PricingModel;
  price: number; // RUB, one-time
  friendPackDiscount: number; // percent
  subscription: SubscriptionPlan;
  demoDay: DemoDay;
  theme: GameTheme;
  status: 'published' | 'draft' | 'hidden' | 'removed';
  createdAt: string;
  stats: { owners: number; subscribers: number; plays: number };
  // viewer context (filled per-request by the API)
  owned: boolean;
  subscribed: boolean;
  canPlayFree: boolean;
}

export type OwnershipType = 'free' | 'purchase' | 'friend-pack' | 'subscription';

export interface Ownership {
  id: string;
  userId: string;
  gameId: string;
  type: OwnershipType;
  price: number;
  giftedBy?: string;
  createdAt: string;
}

export interface SubscriptionRecord {
  id: string;
  userId: string;
  gameId: string;
  developerId: string;
  price: number;
  active: boolean;
  startedAt: string;
}

export interface UserSubscription {
  id: string;
  game: Game;
  expiresAt: string | null; // ISO 8601 or null for legacy
  active: boolean;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export type PaymentKind = 'purchase' | 'friend-pack' | 'subscription';
export type PaymentStatus = 'pending' | 'succeeded' | 'canceled';

/** Mirrors a YooKassa payment object closely enough to swap in the real API. */
export interface Payment {
  id: string;
  gameId: string;
  userId: string;
  kind: PaymentKind;
  amount: number;
  status: PaymentStatus;
  friendUsername?: string;
  confirmationUrl: string; // where YooKassa would redirect the buyer
  createdAt: string;
}

export interface CreateGameInput {
  title: string;
  tagline: string;
  description: string;
  genre: string;
  tags: string[];
  hasBrowserBuild: boolean;
  browserBuildUrl: string | null; // hosted URL alternative to uploading a zip
  hasDownloadBuild: boolean;
  downloadPlatforms: string[];
  supportsMultiplayer: boolean;
  pricingModel: PricingModel;
  price: number;
  friendPackDiscount: number;
  subscription: SubscriptionPlan;
  demoDay: { enabled: boolean; startsAt: string | null; endsAt: string | null };
  theme: GameTheme;
  // real files for multipart upload (browser sends, S3 stores)
  coverFile: File | null;
  backgroundFile: File | null;
  screenshotFiles: File[];
  browserBuildFile: File | null;
  downloadFile: File | null;
}

export interface Report {
  id: string;
  reporterId: string;
  targetType: string;
  targetId: string;
  reason: string;
  details: string;
  status: 'open' | 'resolved' | 'dismissed';
  resolution: string;
  handledBy: string | null;
  createdAt: string;
  resolvedAt: string | null;
}

export interface ServiceSettings {
  commissionPercent: number;
  trendingEnabled: boolean;
  popularEnabled: boolean;
}

export interface HomeSections {
  trending: Game[];
  popular: Game[];
  newest: Game[];
  demoDay: Game[];
}

export interface ListFilters {
  search?: string;
  genre?: string;
  pricing?: '' | 'free' | 'paid' | 'subscription' | 'demo';
  tag?: string;
  sort?: 'new' | 'popular' | 'price-asc' | 'price-desc';
}

export interface ApiError {
  status: number;
  message: string;
}

export interface Payout {
  ID: string;
  DeveloperID: string;
  Amount: number; // kopecks
  Status: 'pending' | 'paid' | 'rejected';
  Note: string;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface PayoutWithDev extends Payout {
  DeveloperUsername: string;
}

export interface PayoutBalance {
  earned: number;    // kopecks — total earned across all completed payments
  available: number; // kopecks — earned minus already requested
  history: Payout[];
}

export interface APIKey {
  id: string;
  name: string;
  createdAt: string;
  lastUsedAt: string | null;
  revoked: boolean;
}
