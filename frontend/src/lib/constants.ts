export const GENRES = [
  'Action',
  'Adventure',
  'Platformer',
  'Puzzle',
  'RPG',
  'Roguelike',
  'Strategy',
  'Shooter',
  'Simulation',
  'Racing',
  'Horror',
  'Visual Novel',
  'Other',
] as const;

export const PLATFORMS = ['Windows', 'macOS', 'Linux'] as const;

export const TAG_SUGGESTIONS = [
  'pixel-art',
  'singleplayer',
  'multiplayer',
  'co-op',
  'retro',
  'atmospheric',
  'difficult',
  'story-rich',
  'procedural',
  'metroidvania',
  'cozy',
  'sci-fi',
  'fantasy',
  'minimalist',
];

// Curated accent presets for the per-game page customiser.
export const ACCENT_PRESETS: { name: string; accent: string; accent2: string; background: string }[] = [
  { name: 'Ember', accent: '#ff6a2c', accent2: '#ffb23e', background: '#140d0a' },
  { name: 'Violet', accent: '#8b5cf6', accent2: '#d946ef', background: '#0f0a16' },
  { name: 'Toxic', accent: '#84cc16', accent2: '#22d3ee', background: '#0a1410' },
  { name: 'Frost', accent: '#38bdf8', accent2: '#818cf8', background: '#0a0f17' },
  { name: 'Rose', accent: '#fb7185', accent2: '#f59e0b', background: '#160a0d' },
  { name: 'Mono', accent: '#e5e5e5', accent2: '#9ca3af', background: '#0c0c0e' },
];

export const RUB = (n: number) =>
  n === 0
    ? 'Free'
    : new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 }).format(n) + ' ₽';
