# Game Development Guide for IndieForge

A practical guide to publishing games on IndieForge — from a simple browser demo
to a full-featured multiplayer title.

---

## Table of contents

1. [Game types](#game-types)
2. [Preparing a browser build](#preparing-a-browser-build)
3. [Uploading your game](#uploading-your-game)
4. [Multiplayer](#multiplayer)
5. [Monetisation](#monetisation)
6. [TODO / future platform features](#todo--future-platform-features)

---

## Game types

IndieForge supports three distribution modes that can be combined:

| Mode | What the player gets | When to use |
|------|---------------------|-------------|
| **Free** | Plays/downloads for free | demos, open-source, jam entries |
| **Paid** | One-time purchase → permanent ownership | full games, DLC |
| **Subscription** | Access via a subscription plan | curated bundles, back-catalogue |

Every game can also carry a **Demo Day** window (a free-play period you set) and
a **Friend Pack** discount (reduced price when a friend already owns the game).

---

## Preparing a browser build

IndieForge plays browser builds directly in an `<iframe>` on the game page.
The build must be a **ZIP archive** containing an `index.html` at the root.

### Vanilla HTML / JS / Canvas

```
mygame.zip
└── index.html      ← entry point (must be at root)
    ├── game.js
    └── assets/
```

The page is sandboxed inside an iframe. Avoid `localStorage` keys that clash
with the host page; use a namespaced prefix like `mygame_`.

### Unity WebGL

1. `File → Build Settings → Platform: WebGL → Switch Platform`
2. `Player Settings → Publishing Settings` — set **Compression Format** to
   *Disabled* or *Gzip* (Brotli needs server-side headers that may not be set).
3. `Build` → Unity outputs a folder like:

   ```
   Build/
   ├── index.html
   ├── Build/
   │   ├── mygame.wasm
   │   ├── mygame.js
   │   └── mygame.data
   └── TemplateData/
   ```

4. ZIP the entire output folder **with `index.html` at the root** of the archive.
5. Upload as **Browser build** on the Create Game page.

> **Size note:** a minimal Unity WebGL build is 5–15 MB. Large builds work fine
> but take longer to load in the browser; consider a loading screen.

### Godot 4 (HTML5 export)

1. `Project → Export → Add → Web`
2. Enable **Export PCK/ZIP** → export to a folder
3. ZIP the output with `index.html` at the root

### Phaser / other JS frameworks

Build your project (`npm run build`), then ZIP the `dist/` folder (rename
`dist/index.html` → root `index.html` if needed).

---

## Uploading your game

| Field | Notes |
|-------|-------|
| **Cover image** | Shown in the catalog. Recommended 460 × 215 px (Steam-style landscape). |
| **Screenshots** | Up to 8. Shown below the game on the game page. |
| **Wallpaper** | Optional background image shown behind the game page mat. Wide/cinematic images work best. |
| **Browser build** | ZIP archive described above. Scanned by ClamAV on upload. |
| **Downloadable build** | Any file (installer, `.zip`, `.apk`, …). Served via a short-lived presigned URL — never publicly listed. |
| **Mat color** | Background colour of the center content column on the game page. |
| **Accent colors** | Used for buttons, tags, and UI highlights on the game page. |

---

## Multiplayer

### Same-device / same-browser (tabs)

Use the [BroadcastChannel API](https://developer.mozilla.org/en-US/docs/Web/API/BroadcastChannel):

```js
const ch = new BroadcastChannel('mygame');
ch.postMessage({ type: 'move', x: player.x, y: player.y });
ch.onmessage = ({ data }) => { /* handle */ };
```

Zero infrastructure, works offline, but limited to tabs in the same browser
on the same device.

### Real cross-device multiplayer — Unity + Unity Relay (recommended)

For real networked multiplayer the recommended stack is:

- **[Unity Netcode for GameObjects](https://docs-multiplayer.unity3d.com/netcode/current/about/)** — high-level networking (object sync, RPCs, ownership)
- **[Unity Relay](https://docs.unity.com/ugs/manual/relay/manual/introduction)** — NAT traversal relay hosted by Unity; no server infra needed
- **[Unity Lobby](https://docs.unity.com/ugs/manual/lobby/manual/unity-lobby-service)** — room discovery and join codes

All three are part of **Unity Gaming Services (UGS)** and have a generous free
tier (Relay: up to 10 CCU free, Lobby: up to 250 concurrent lobbies free).

**Setup in brief:**

1. Create a project at [dashboard.unity.com](https://dashboard.unity.com).
2. Enable Relay, Lobby, and Authentication in your UGS project.
3. In Unity: `Window → Package Manager` → install
   *Netcode for GameObjects*, *Relay*, *Lobby*, *Authentication*.
4. In your game:

```csharp
await UnityServices.InitializeAsync();
await AuthenticationService.Instance.SignInAnonymouslyAsync();

// Host: create a relay allocation and a lobby
var alloc = await RelayService.Instance.CreateAllocationAsync(maxPlayers: 4);
string joinCode = await RelayService.Instance.GetJoinCodeAsync(alloc.AllocationId);
// share joinCode with players via lobby or out-of-band

// Client: join by code
var joinAlloc = await RelayService.Instance.JoinAllocationAsync(joinCode);
```

5. Pass the allocation data to `NetworkManager` as the transport, then use
   standard Netcode APIs (`NetworkVariable`, `ServerRpc`, `ClientRpc`).

The build is exported as WebGL and uploaded to IndieForge exactly like any
other browser build — Unity Relay handles all connectivity transparently.

### Phaser / vanilla JS

For non-Unity games that need multiplayer, options include:

- **[Colyseus](https://colyseus.io/)** — open-source authoritative game server with a JS client; free self-hosting or cloud plan
- **[Ably](https://ably.com/)** / **[PubNub](https://www.pubnub.com/)** — managed pub/sub with free tiers
- **[Supabase Realtime](https://supabase.com/realtime)** — WebSocket broadcast on top of Postgres; generous free tier
- **WebRTC data channels** (via [PeerJS](https://peerjs.com/) or [simple-peer](https://github.com/feross/simple-peer)) — peer-to-peer, no relay server costs

---

## Monetisation

### Paid game

Set a price in the *Pricing* section when creating the game. Players purchase
once and get permanent ownership. Payments are processed by YooKassa; the
platform takes a commission (set by the admin, visible in the purchase flow).

### Subscription plan

Games can be included in subscription plans created by you or the platform
admin. Players with an active subscription get access without a separate
purchase.

### Friend Pack

Enable *Friend Pack* and set a discounted price. A player whose friend already
owns the game sees the lower price automatically at checkout.

### Demo Day

Set start and end dates for a free-play window. During Demo Day, any visitor
can play the browser build without purchasing — useful for launch visibility.

---

## TODO / future platform features

Ideas and potential improvements not yet implemented:

- **Self-hosted WebSocket relay** — a lightweight room-based message relay
  built into the IndieForge backend, for non-Unity games that need simple
  cross-device multiplayer without third-party services. Would expose
  `GET /api/ws/relay?room=<code>` and relay messages between two (or N)
  players in the same room. Low complexity, no game logic server-side —
  worth considering once there is demand from non-Unity developers.

- **Game API token** — a signed token injected into the iframe so a game can
  call `/api/me` and know which IndieForge user is playing (display name,
  ownership, subscription status).

- **Leaderboards API** — platform-level high-score storage so any game can
  `POST /api/leaderboard/:gameId` without rolling its own backend.

- **Save data API** — small cloud key-value store per user per game, replacing
  `localStorage` for cross-device saves.

- **Achievement system** — badge definitions stored per game, granted via API
  call from the game client, displayed on the user's profile.

- **In-game purchases** — micro-transaction support: game client calls
  `/api/payments/ingame`, player approves in an overlay, game client gets a
  signed receipt. Would use the existing YooKassa integration.
