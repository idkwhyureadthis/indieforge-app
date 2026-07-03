import { Link } from 'react-router-dom';
import { BookOpen, Gamepad2, Globe, DollarSign, Wifi, Lightbulb, Package, ChevronRight, Key } from 'lucide-react';

function Section({ id, icon: Icon, title, children }: {
  id: string; icon: typeof BookOpen; title: string; children: React.ReactNode;
}) {
  return (
    <section id={id} className="scroll-mt-24">
      <div className="mb-6 flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-ember-500/15 text-ember-400">
          <Icon className="h-5 w-5" />
        </div>
        <h2 className="font-display text-xl font-700 text-mist-50">{title}</h2>
      </div>
      <div className="space-y-4 text-mist-300 leading-relaxed">{children}</div>
    </section>
  );
}

function Code({ children }: { children: string }) {
  return (
    <pre className="overflow-x-auto rounded-xl bg-iron-900 border border-iron-700 p-4 text-sm text-mist-200 font-mono leading-relaxed">
      {children}
    </pre>
  );
}

function Badge({ children, color = 'iron' }: { children: React.ReactNode; color?: string }) {
  const colors: Record<string, string> = {
    iron: 'bg-iron-700 text-mist-300',
    ember: 'bg-ember-500/20 text-ember-300',
    green: 'bg-emerald-500/15 text-emerald-300',
    blue: 'bg-sky-500/15 text-sky-300',
  };
  return (
    <span className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-600 ${colors[color]}`}>
      {children}
    </span>
  );
}

function Table({ headers, rows }: { headers: string[]; rows: string[][] }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-iron-700">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-iron-700 bg-iron-800/50">
            {headers.map(h => (
              <th key={h} className="px-4 py-3 text-left text-xs font-600 uppercase tracking-wider text-mist-400">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className={i % 2 === 0 ? 'bg-iron-900' : 'bg-iron-800/30'}>
              {row.map((cell, j) => (
                <td key={j} className="px-4 py-3 text-mist-300">{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

const TOC = [
  { id: 'game-types',     icon: Gamepad2,  label: 'Game types' },
  { id: 'browser-build',  icon: Globe,     label: 'Browser build' },
  { id: 'uploading',      icon: Package,   label: 'Uploading your game' },
  { id: 'multiplayer',    icon: Wifi,      label: 'Multiplayer' },
  { id: 'monetisation',   icon: DollarSign, label: 'Monetisation' },
  { id: 'developer-api',  icon: Key,        label: 'Developer API' },
  { id: 'roadmap',        icon: Lightbulb,  label: 'Roadmap' },
];

export function DocsPage() {
  return (
    <div className="container-page py-12">
      <div className="mb-10">
        <p className="mb-2 text-sm font-600 uppercase tracking-widest text-ember-400">Developer docs</p>
        <h1 className="font-display text-3xl font-800 text-mist-50">Publishing games on IndieForge</h1>
        <p className="mt-3 max-w-2xl text-mist-400">
          Everything you need to go from a local build to a live game — browser play, downloads,
          multiplayer, and monetisation.
        </p>
      </div>

      <div className="flex flex-col gap-12 lg:flex-row lg:gap-16">

        {/* sidebar TOC */}
        <aside className="hidden lg:block lg:w-52 shrink-0">
          <nav className="sticky top-24 space-y-1">
            {TOC.map(({ id, icon: Icon, label }) => (
              <a
                key={id}
                href={`#${id}`}
                className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-mist-400 hover:bg-iron-800 hover:text-mist-100 transition-colors"
              >
                <Icon className="h-4 w-4 shrink-0" />
                {label}
              </a>
            ))}
          </nav>
        </aside>

        {/* content */}
        <div className="min-w-0 flex-1 space-y-14">

          {/* game types */}
          <Section id="game-types" icon={Gamepad2} title="Game types">
            <p>Each game can combine multiple distribution modes.</p>
            <Table
              headers={['Mode', 'What the player gets', 'When to use']}
              rows={[
                ['Free', 'Plays / downloads at no cost', 'Demos, jam entries, open-source'],
                ['Paid', 'One-time purchase → permanent ownership', 'Full games, DLC, assets'],
                ['Subscription', 'Access via a subscription plan', 'Curated bundles, back-catalogue'],
              ]}
            />
            <p>
              Every game can also define a <strong className="text-mist-100">Demo Day</strong> (a
              time-limited free-play window) and a{' '}
              <strong className="text-mist-100">Friend Pack</strong> discount applied automatically
              when a friend already owns the game.
            </p>
          </Section>

          {/* browser build */}
          <Section id="browser-build" icon={Globe} title="Preparing a browser build">
            <p>
              IndieForge embeds browser builds in an <code className="rounded bg-iron-800 px-1.5 py-0.5 text-xs text-mist-200">&lt;iframe&gt;</code>.
              The build must be a <strong className="text-mist-100">ZIP archive with <code className="rounded bg-iron-800 px-1.5 py-0.5 text-xs text-mist-200">index.html</code> at the root</strong>.
            </p>

            <div className="space-y-8 pt-2">
              <div>
                <div className="mb-3 flex items-center gap-2">
                  <span className="font-600 text-mist-100">Unity WebGL</span>
                  <Badge color="blue">Recommended</Badge>
                </div>
                <ol className="space-y-2 pl-4 text-sm list-decimal marker:text-mist-500">
                  <li><code className="text-mist-200">File → Build Settings → Platform: WebGL → Switch Platform</code></li>
                  <li>
                    <code className="text-mist-200">Player Settings → Publishing Settings</code> — set{' '}
                    <strong className="text-mist-100">Compression Format</strong> to{' '}
                    <em>Disabled</em> or <em>Gzip</em> (Brotli requires server headers that may not be set).
                  </li>
                  <li><code className="text-mist-200">Build</code> → ZIP the output folder so <code className="text-mist-200">index.html</code> is at the archive root.</li>
                  <li>Upload as <strong className="text-mist-100">Browser build</strong> on the create game page.</li>
                </ol>
                <p className="mt-3 text-sm text-mist-500">
                  A minimal Unity WebGL build is 5–15 MB. Large builds work fine but take longer to
                  load — consider a loading screen.
                </p>
              </div>

              <div>
                <p className="mb-3 font-600 text-mist-100">Godot 4 (HTML5 export)</p>
                <ol className="space-y-2 pl-4 text-sm list-decimal marker:text-mist-500">
                  <li><code className="text-mist-200">Project → Export → Add → Web</code></li>
                  <li>Export to a folder, then ZIP with <code className="text-mist-200">index.html</code> at the root.</li>
                </ol>
              </div>

              <div>
                <p className="mb-3 font-600 text-mist-100">Phaser / Vanilla JS / Canvas</p>
                <Code>{`mygame.zip
└── index.html      ← must be at archive root
    ├── game.js
    └── assets/`}</Code>
                <p className="mt-3 text-sm text-mist-500">
                  The page runs sandboxed in an iframe. Namespace your{' '}
                  <code className="text-mist-200">localStorage</code> keys (e.g.{' '}
                  <code className="text-mist-200">mygame_save</code>) to avoid conflicts.
                </p>
              </div>
            </div>
          </Section>

          {/* uploading */}
          <Section id="uploading" icon={Package} title="Uploading your game">
            <Table
              headers={['Field', 'Notes']}
              rows={[
                ['Cover image', 'Shown in the catalog. 460 × 215 px landscape recommended.'],
                ['Screenshots', 'Up to 8. Shown below the game on its page.'],
                ['Wallpaper', 'Background image behind the game page. Wide / cinematic images work best.'],
                ['Mat color', 'Background colour of the centre content column on the game page.'],
                ['Accent colors', 'Used for buttons, tags, and highlights on the game page.'],
                ['Browser build', 'ZIP archive — see above. Scanned by antivirus on upload.'],
                ['Downloadable build', 'Any file (.zip, installer, .apk, …). Served via a short-lived private link.'],
              ]}
            />
            <p>
              You can update any of these fields later from your{' '}
              <Link to="/dashboard" className="text-ember-400 hover:underline">Studio dashboard</Link>.
            </p>
          </Section>

          {/* multiplayer */}
          <Section id="multiplayer" icon={Wifi} title="Multiplayer">

            <div className="space-y-8">
              <div>
                <div className="mb-2 flex items-center gap-2">
                  <p className="font-600 text-mist-100">Same browser / same device</p>
                  <Badge>BroadcastChannel</Badge>
                </div>
                <p className="text-sm">Zero infrastructure, works offline, no server required. Limited to tabs in the same browser on the same device.</p>
                <Code>{`const ch = new BroadcastChannel('mygame');
ch.postMessage({ type: 'move', x: player.x, y: player.y });
ch.onmessage = ({ data }) => { /* handle */ };`}</Code>
              </div>

              <div>
                <div className="mb-2 flex items-center gap-2">
                  <p className="font-600 text-mist-100">Real cross-device multiplayer — Unity + Unity Relay</p>
                  <Badge color="ember">Recommended</Badge>
                </div>
                <p className="text-sm mb-4">
                  For genuine networked multiplayer the recommended stack is{' '}
                  <strong className="text-mist-100">Unity Gaming Services (UGS)</strong>:
                </p>
                <ul className="space-y-2 pl-4 text-sm list-disc marker:text-ember-500">
                  <li>
                    <strong className="text-mist-100">Netcode for GameObjects</strong> — high-level sync (NetworkVariable, RPCs, ownership)
                  </li>
                  <li>
                    <strong className="text-mist-100">Unity Relay</strong> — NAT traversal relay hosted by Unity; no server infra needed.{' '}
                    <span className="text-mist-500">Free up to 10 CCU.</span>
                  </li>
                  <li>
                    <strong className="text-mist-100">Unity Lobby</strong> — room discovery and join codes.{' '}
                    <span className="text-mist-500">Free up to 250 concurrent lobbies.</span>
                  </li>
                </ul>

                <div className="mt-4 rounded-xl border border-iron-700 bg-iron-900/60 p-4 text-sm">
                  <p className="mb-3 font-600 text-mist-200">Quick setup</p>
                  <ol className="space-y-1.5 pl-4 list-decimal marker:text-mist-500">
                    <li>Create a project at <span className="text-mist-200">dashboard.unity.com</span> and enable Relay, Lobby, and Authentication.</li>
                    <li>In Unity: <code className="text-mist-200">Window → Package Manager</code> → install <em>Netcode for GameObjects</em>, <em>Relay</em>, <em>Lobby</em>, <em>Authentication</em>.</li>
                    <li>Export as <strong className="text-mist-100">WebGL</strong> and upload to IndieForge as a browser build — Unity Relay handles connectivity transparently.</li>
                  </ol>
                </div>

                <Code>{`await UnityServices.InitializeAsync();
await AuthenticationService.Instance.SignInAnonymouslyAsync();

// Host — create relay allocation + lobby
var alloc = await RelayService.Instance.CreateAllocationAsync(maxPlayers: 4);
string joinCode = await RelayService.Instance.GetJoinCodeAsync(alloc.AllocationId);
// share joinCode with your players (e.g. display it in-game)

// Client — join by code
var joinAlloc = await RelayService.Instance.JoinAllocationAsync(joinCode);`}</Code>
              </div>

              <div>
                <div className="mb-2 flex items-center gap-2">
                  <p className="font-600 text-mist-100">Phaser / vanilla JS alternatives</p>
                </div>
                <div className="grid gap-3 sm:grid-cols-2 text-sm">
                  {[
                    { name: 'Colyseus', desc: 'Open-source authoritative game server + JS client. Free self-hosting or cloud.' },
                    { name: 'Ably / PubNub', desc: 'Managed pub/sub WebSocket with generous free tiers.' },
                    { name: 'Supabase Realtime', desc: 'WebSocket broadcast on top of Postgres. Free tier.' },
                    { name: 'PeerJS / simple-peer', desc: 'WebRTC data channels, true peer-to-peer — no relay server costs.' },
                  ].map(({ name, desc }) => (
                    <div key={name} className="rounded-xl border border-iron-700 bg-iron-800/40 p-4">
                      <p className="mb-1 font-600 text-mist-200">{name}</p>
                      <p className="text-mist-400">{desc}</p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </Section>

          {/* monetisation */}
          <Section id="monetisation" icon={DollarSign} title="Monetisation">
            <div className="grid gap-4 sm:grid-cols-2">
              {[
                {
                  title: 'Paid game',
                  body: 'Set a price when creating the game. Players purchase once and get permanent ownership. The platform takes a commission visible in the purchase flow.',
                },
                {
                  title: 'Subscription plan',
                  body: 'Include your game in a subscription plan. Players with an active subscription get access without a separate purchase.',
                },
                {
                  title: 'Friend Pack',
                  body: 'Enable Friend Pack and set a discounted price. A buyer whose friend already owns the game sees the lower price at checkout automatically.',
                },
                {
                  title: 'Demo Day',
                  body: 'Set start and end dates for a free-play window. During Demo Day any visitor can play the browser build — useful for launch visibility.',
                },
              ].map(({ title, body }) => (
                <div key={title} className="rounded-xl border border-iron-700 bg-iron-800/40 p-5">
                  <p className="mb-2 font-600 text-mist-100">{title}</p>
                  <p className="text-sm text-mist-400">{body}</p>
                </div>
              ))}
            </div>
            <p className="text-sm text-mist-500">
              Payments are processed by YooKassa. Funds are credited to your YooKassa account
              minus the platform commission.
            </p>
          </Section>

          {/* developer api */}
          <Section id="developer-api" icon={Key} title="Developer API">
            <p>
              The Developer API lets your game server (or a serverless function) verify whether a
              player has an active subscription — so you can grant in-game bonuses without trusting
              the client.
            </p>

            <h3 className="font-600 text-mist-100">Authentication</h3>
            <p>
              Create an API key in your <Link to="/dashboard" className="text-ember-400 hover:underline">Studio dashboard</Link>.
              Pass it in every request as the <code className="rounded bg-iron-800 px-1 text-sm text-mist-200">X-API-Key</code> header.
              Keys start with <code className="rounded bg-iron-800 px-1 text-sm text-mist-200">sk_</code> and are stored hashed — copy them when created.
            </p>

            <h3 className="font-600 text-mist-100">Verify a subscription</h3>
            <Code>{`GET https://indieforge.example.com/api/v1/subscriptions/verify
    ?gameId=<game-id-or-slug>
    &userId=<player-user-id>

X-API-Key: sk_xxxxxxxxxxxxxxxx...

HTTP 200
{
  "subscribed": true,
  "expiresAt": "2026-07-25T01:34:12Z"   // null for legacy subs
}`}</Code>

            <h3 className="font-600 text-mist-100">For browser games — direct subscription check</h3>
            <p>
              If your game runs entirely in the browser (HTML5 / WebGL), the player is already logged
              into IndieForge and their bearer token is available in{' '}
              <code className="rounded bg-iron-800 px-1 text-sm text-mist-200">localStorage</code>.
              You can call the user-auth endpoint directly — no API key needed:
            </p>
            <Code>{`// Inside your browser game JS (no backend required)
const token = localStorage.getItem('indieforge_token');
const res   = await fetch('/api/me/subscription-status?gameId=my-game-slug', {
  headers: { Authorization: \`Bearer \${token}\` },
});
const { subscribed, expiresAt } = await res.json();
if (subscribed) unlockPremiumContent();`}</Code>

            <h3 className="font-600 text-mist-100">For downloadable games — launch tokens</h3>
            <p>
              A downloaded binary has no browser session, so the player can't share a bearer token with the
              game. Use <strong>launch tokens</strong> instead — one-time tokens generated on the game page.
            </p>
            <div className="space-y-2 my-2">
              {[
                ['1. Player generates a token', 'On the IndieForge game page, the player clicks "Get launch token". A 15-minute token (lt_…) appears. They copy it.'],
                ['2. Game reads the token', 'At startup the game asks the player to paste it — via a text prompt, dialog box, or CLI flag.'],
                ['3. Your backend verifies it', 'Send the token to your server, which calls POST /api/v1/launch-tokens/verify with your API key. The token is deleted on first use (replay-safe).'],
                ['4. You receive player identity', 'The response includes userId, gameId, subscribed, and expiresAt. Cache it for the session duration.'],
              ].map(([title, body]) => (
                <div key={title as string} className="flex gap-3 rounded-xl border border-iron-700/60 bg-iron-800/30 p-4">
                  <ChevronRight className="mt-0.5 h-4 w-4 shrink-0 text-sky-400" />
                  <div>
                    <p className="font-600 text-mist-200">{title}</p>
                    <p className="mt-0.5 text-sm text-mist-400">{body}</p>
                  </div>
                </div>
              ))}
            </div>
            <Code>{`POST /api/v1/launch-tokens/verify
X-API-Key: sk_xxxxxxxxxxxxxxxx...
Content-Type: application/json

{ "token": "lt_<token-the-player-pasted>" }

// 200 OK
{
  "userId":     "usr_abc123",
  "gameId":     "gme_xyz789",
  "subscribed": true,
  "expiresAt":  "2026-07-25T01:34:12Z"   // null if no auto-renewal expiry
}`}</Code>

            <h3 className="font-600 text-mist-100">Getting the player's userId (server-side verify)</h3>
            <p>
              When calling <code className="rounded bg-iron-800 px-1 text-sm text-mist-200">GET /api/v1/subscriptions/verify</code> you must
              supply the player's IndieForge <code className="rounded bg-iron-800 px-1 text-sm text-mist-200">userId</code>.
              For browser games read it from <code className="rounded bg-iron-800 px-1 text-sm text-mist-200">GET /api/me</code> using the
              player's bearer token. For downloadable games use launch tokens — they return the userId directly.
            </p>

            <h3 className="font-600 text-mist-100">Security model</h3>
            <div className="space-y-2">
              {[
                ['API key is server-side only', 'Never embed sk_ keys in browser-side JS — your backend calls the verify endpoint, not the game client.'],
                ['Scoped to your games', 'A key can only verify subscriptions to games you own. Querying another developer\'s game returns subscribed: false.'],
                ['Rate limit: 60 req / min per key', 'Requests beyond the limit get HTTP 429. Cache results for a few minutes to stay well within limits.'],
                ['Key revocation', 'Revoke a key instantly from your dashboard if it is compromised — no propagation delay.'],
              ].map(([title, body]) => (
                <div key={title as string} className="flex gap-3 rounded-xl border border-iron-700/60 bg-iron-800/30 p-4">
                  <ChevronRight className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />
                  <div>
                    <p className="font-600 text-mist-200">{title}</p>
                    <p className="mt-0.5 text-sm text-mist-400">{body}</p>
                  </div>
                </div>
              ))}
            </div>

            <h3 className="font-600 text-mist-100">Example: Node.js server</h3>
            <Code>{`// server.js — runs on YOUR backend, not in the browser
app.get('/api/bonus-items', async (req, res) => {
  const userId = req.session.indieforgeUserId; // from your login flow

  const r = await fetch(
    \`https://indieforge.example.com/api/v1/subscriptions/verify\` +
    \`?gameId=my-awesome-game&userId=\${userId}\`,
    { headers: { 'X-API-Key': process.env.INDIEFORGE_API_KEY } }
  );
  const { subscribed } = await r.json();

  res.json({ items: subscribed ? SUBSCRIBER_ITEMS : FREE_ITEMS });
});`}</Code>

            <Table
              headers={['Response field', 'Type', 'Description']}
              rows={[
                ['subscribed', 'boolean', 'true if the user has an active subscription to the requested game'],
                ['expiresAt', 'string | null', 'ISO 8601 UTC timestamp of the next renewal date; null for legacy subs without auto-renewal'],
              ]}
            />
          </Section>

          {/* roadmap */}
          <Section id="roadmap" icon={Lightbulb} title="Roadmap">
            <p>Features being considered for future platform versions.</p>
            <div className="space-y-3">
              {[
                {
                  title: 'Self-hosted WebSocket relay',
                  body: 'A lightweight room-based message relay built into the IndieForge backend — for non-Unity games that need simple cross-device multiplayer without third-party services.',
                },
                {
                  title: 'Leaderboards',
                  body: 'Platform-level high-score storage — any game can POST a score without rolling its own backend.',
                },
                {
                  title: 'Cloud save',
                  body: 'Small key-value store per user per game, replacing localStorage for cross-device saves.',
                },
                {
                  title: 'Achievements',
                  body: 'Badge definitions stored per game, granted via API from the game client, displayed on the player\'s profile.',
                },
              ].map(({ title, body }) => (
                <div key={title} className="flex gap-4 rounded-xl border border-iron-700/60 bg-iron-800/30 p-4">
                  <ChevronRight className="mt-0.5 h-4 w-4 shrink-0 text-ember-500" />
                  <div>
                    <p className="font-600 text-mist-200">{title}</p>
                    <p className="mt-0.5 text-sm text-mist-400">{body}</p>
                  </div>
                </div>
              ))}
            </div>
          </Section>

        </div>
      </div>
    </div>
  );
}
