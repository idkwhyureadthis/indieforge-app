import { useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Check, ChevronLeft, ChevronRight, Globe, Download, Users2, X, Upload, Image as ImageIcon,
  Repeat, Gift, CalendarClock, Sparkles, Plus, Rocket,
} from 'lucide-react';
import type { CreateGameInput, GameTheme } from '@/lib/types';
import { api, ApiError } from '@/lib/api';
import { CoverArt } from '@/components/CoverArt';
import { Spinner } from '@/components/ui';
import { ACCENT_PRESETS, GENRES, PLATFORMS, RUB, TAG_SUGGESTIONS } from '@/lib/constants';
import { bytesToMB } from '@/lib/files';
import { useToast } from '@/context/ToastContext';

const STEPS = ['Basics', 'Builds', 'Pricing', 'Page style', 'Media', 'Review'] as const;

const initialTheme: GameTheme = {
  accent: ACCENT_PRESETS[0].accent,
  accent2: ACCENT_PRESETS[0].accent2,
  background: ACCENT_PRESETS[0].background,
  layout: 'immersive',
  cardShape: 'rounded',
};

const initial: CreateGameInput = {
  title: '',
  tagline: '',
  description: '',
  genre: 'Action',
  tags: [],
  hasBrowserBuild: false,
  browserBuildUrl: null,
  hasDownloadBuild: false,
  downloadPlatforms: [],
  supportsMultiplayer: false,
  pricingModel: 'free',
  price: 0,
  friendPackDiscount: 0,
  subscription: { enabled: false, price: 149, period: 'month', benefits: [''], chatLink: '' },
  demoDay: { enabled: false, startsAt: null, endsAt: null },
  theme: initialTheme,
  coverFile: null,
  backgroundFile: null,
  screenshotFiles: [],
  browserBuildFile: null,
  downloadFile: null,
};

export function CreateGamePage() {
  const navigate = useNavigate();
  const toast = useToast();
  const [step, setStep] = useState(0);
  const [form, setForm] = useState<CreateGameInput>(initial);
  const [tagDraft, setTagDraft] = useState('');
  const [publishing, setPublishing] = useState(false);

  const update = (patch: Partial<CreateGameInput>) => setForm((f) => ({ ...f, ...patch }));

  const stepError = useMemo(() => validateStep(step, form), [step, form]);

  const next = () => {
    if (stepError) {
      toast(stepError, 'error');
      return;
    }
    setStep((s) => Math.min(s + 1, STEPS.length - 1));
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };
  const back = () => {
    setStep((s) => Math.max(s - 1, 0));
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const publish = async () => {
    setPublishing(true);
    try {
      const game = await api.createGame({
        ...form,
        subscription: {
          ...form.subscription,
          benefits: form.subscription.benefits.map((b) => b.trim()).filter(Boolean),
        },
      });
      toast('Published! Your game is live.', 'success');
      navigate(`/game/${game.slug}`);
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Could not publish', 'error');
      setPublishing(false);
    }
  };

  return (
    <div className="container-page max-w-5xl py-8">
      <h1 className="text-3xl font-700 text-mist-50">Upload a game</h1>
      <p className="mt-1 text-mist-400">Forge a new listing — browser or downloadable, free or paid.</p>

      <Stepper step={step} />

      <div className="mt-8 grid gap-8 lg:grid-cols-[1fr_19rem]">
        <div className="card p-6">
          {step === 0 && <BasicsStep form={form} update={update} tagDraft={tagDraft} setTagDraft={setTagDraft} />}
          {step === 1 && <BuildsStep form={form} update={update} />}
          {step === 2 && <PricingStep form={form} update={update} />}
          {step === 3 && <StyleStep form={form} update={update} />}
          {step === 4 && <MediaStep form={form} update={update} />}
          {step === 5 && <ReviewStep form={form} />}

          <div className="mt-8 flex items-center justify-between border-t border-iron-700 pt-5">
            <button onClick={back} disabled={step === 0} className="btn btn-ghost disabled:opacity-40">
              <ChevronLeft className="h-4 w-4" /> Back
            </button>
            {step < STEPS.length - 1 ? (
              <button onClick={next} className="btn btn-primary">
                Continue <ChevronRight className="h-4 w-4" />
              </button>
            ) : (
              <button onClick={publish} disabled={publishing} className="btn btn-primary btn-lg">
                {publishing ? <Spinner /> : <><Rocket className="h-4 w-4" /> Publish game</>}
              </button>
            )}
          </div>
        </div>

        {/* Live preview */}
        <aside className="lg:sticky lg:top-20 lg:self-start">
          <p className="mb-2 text-xs font-600 uppercase tracking-wide text-mist-500">Live preview</p>
          <PreviewCard form={form} />
        </aside>
      </div>
    </div>
  );
}

function validateStep(step: number, f: CreateGameInput): string | null {
  if (step === 0) {
    if (!f.title.trim()) return 'Give your game a title';
    if (!f.tagline.trim()) return 'Add a short tagline';
  }
  if (step === 1 && !f.hasBrowserBuild && !f.hasDownloadBuild)
    return 'Add at least one build: browser or downloadable';
  if (step === 1 && f.hasBrowserBuild && !f.browserBuildUrl && !f.browserBuildFile)
    return 'Provide a browser build URL or upload an HTML build';
  if (step === 1 && f.hasBrowserBuild && f.browserBuildUrl) {
    try {
      const u = new URL(f.browserBuildUrl);
      const host = u.hostname.toLowerCase();
      const ownHost = window.location.hostname.toLowerCase();
      if (host === 'localhost' || host === '127.0.0.1' || host === ownHost)
        return 'Hosted build URL cannot point to IndieForge itself or localhost';
      if (u.protocol !== 'https:' && !(host === 'localhost' || host === '127.0.0.1'))
        return 'Hosted build URL must use HTTPS';
    } catch {
      return 'Hosted build URL is not a valid URL';
    }
  }
  if (step === 1 && f.hasDownloadBuild && !f.downloadFile)
    return 'Upload a downloadable build file';
  if (step === 2 && f.pricingModel === 'paid' && f.price <= 0)
    return 'Set a price above zero, or switch to Free';
  if (step === 2 && f.subscription.enabled && f.subscription.price <= 0)
    return 'Set a subscription price';
  return null;
}

// ---------------------------------------------------------------------------
// Stepper
// ---------------------------------------------------------------------------
function Stepper({ step }: { step: number }) {
  return (
    <div className="mt-6 flex items-center gap-1 overflow-x-auto pb-1">
      {STEPS.map((label, i) => {
        const done = i < step;
        const active = i === step;
        return (
          <div key={label} className="flex items-center">
            <div
              className={`flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-500 transition-colors ${
                active ? 'bg-ember-500/15 text-ember-400' : done ? 'text-emerald-400' : 'text-mist-500'
              }`}
            >
              <span
                className={`flex h-5 w-5 items-center justify-center rounded-full text-[11px] font-700 ${
                  active ? 'bg-ember-500 text-iron-950' : done ? 'bg-emerald-500 text-iron-950' : 'bg-iron-700 text-mist-400'
                }`}
              >
                {done ? <Check className="h-3 w-3" /> : i + 1}
              </span>
              <span className="hidden sm:inline">{label}</span>
            </div>
            {i < STEPS.length - 1 && <div className="mx-0.5 h-px w-3 bg-iron-700 sm:w-5" />}
          </div>
        );
      })}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Small reusable bits
// ---------------------------------------------------------------------------
function Toggle({
  checked,
  onChange,
  label,
  description,
  icon: Icon,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: string;
  description?: string;
  icon?: typeof Globe;
}) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      className={`flex w-full cursor-pointer items-center gap-3 rounded-xl border p-3.5 text-left transition-colors ${
        checked ? 'border-ember-500/50 bg-ember-500/5' : 'border-iron-700 bg-iron-900/40 hover:border-iron-600'
      }`}
    >
      {Icon && <Icon className={`h-5 w-5 shrink-0 ${checked ? 'text-ember-400' : 'text-mist-400'}`} />}
      <div className="min-w-0 flex-1">
        <p className="text-sm font-600 text-mist-100">{label}</p>
        {description && <p className="text-xs text-mist-400">{description}</p>}
      </div>
      <span
        className={`relative h-6 w-11 shrink-0 rounded-full transition-colors ${checked ? 'bg-ember-500' : 'bg-iron-600'}`}
      >
        <span
          className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${checked ? 'translate-x-5' : 'translate-x-0.5'}`}
        />
      </span>
    </button>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="label">{label}</label>
      {children}
      {hint && <p className="mt-1 text-xs text-mist-500">{hint}</p>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 1 — Basics
// ---------------------------------------------------------------------------
function BasicsStep({
  form,
  update,
  tagDraft,
  setTagDraft,
}: {
  form: CreateGameInput;
  update: (p: Partial<CreateGameInput>) => void;
  tagDraft: string;
  setTagDraft: (v: string) => void;
}) {
  const addTag = (raw: string) => {
    const t = raw.trim().toLowerCase().replace(/\s+/g, '-');
    if (t && !form.tags.includes(t) && form.tags.length < 8) update({ tags: [...form.tags, t] });
    setTagDraft('');
  };
  return (
    <div className="space-y-5">
      <h2 className="text-lg font-700 text-mist-50">The basics</h2>
      <Field label="Title">
        <input className="input" value={form.title} onChange={(e) => update({ title: e.target.value })} placeholder="Emberfall" maxLength={60} />
      </Field>
      <Field label="Tagline" hint="One punchy line shown on cards and search results.">
        <input className="input" value={form.tagline} onChange={(e) => update({ tagline: e.target.value })} placeholder="A hand-forged roguelike where every run reshapes the underworld." maxLength={120} />
      </Field>
      <div className="grid gap-5 sm:grid-cols-2">
        <Field label="Genre">
          <select className="input" value={form.genre} onChange={(e) => update({ genre: e.target.value })}>
            {GENRES.map((g) => (
              <option key={g} value={g}>
                {g}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Tags" hint="Up to 8. Press Enter to add.">
          <input
            className="input"
            value={tagDraft}
            onChange={(e) => setTagDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                addTag(tagDraft);
              }
            }}
            placeholder="pixel-art, co-op…"
          />
        </Field>
      </div>
      {form.tags.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {form.tags.map((t) => (
            <span key={t} className="chip">
              #{t}
              <button onClick={() => update({ tags: form.tags.filter((x) => x !== t) })} className="ml-1 cursor-pointer text-mist-500 hover:text-rose-400" aria-label={`Remove ${t}`}>
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      )}
      <div className="flex flex-wrap gap-1.5">
        {TAG_SUGGESTIONS.filter((t) => !form.tags.includes(t))
          .slice(0, 8)
          .map((t) => (
            <button key={t} onClick={() => addTag(t)} className="cursor-pointer rounded-full border border-dashed border-iron-600 px-2.5 py-1 text-xs text-mist-400 hover:border-ember-500/60 hover:text-ember-400">
              + {t}
            </button>
          ))}
      </div>
      <Field label="Description" hint="Markdown-ish. Line breaks are preserved.">
        <textarea className="input min-h-40 resize-y" value={form.description} onChange={(e) => update({ description: e.target.value })} placeholder="Tell players what makes your game special…" />
      </Field>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 2 — Builds
// ---------------------------------------------------------------------------
function BuildsStep({ form, update }: { form: CreateGameInput; update: (p: Partial<CreateGameInput>) => void }) {
  const htmlRef = useRef<HTMLInputElement>(null);
  const dlRef = useRef<HTMLInputElement>(null);
  const toast = useToast();

  const pickHtml = (file?: File) => {
    if (!file) return;
    update({ browserBuildFile: file });
    toast(`Browser build “${file.name}” attached`, 'success');
  };
  const pickDownload = (file?: File) => {
    if (!file) return;
    update({ downloadFile: file });
    toast(`Build “${file.name}” attached`, 'success');
  };

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-700 text-mist-50">Builds & how players run it</h2>
      <p className="-mt-3 text-sm text-mist-400">Add at least one. Like itch.io, you can ship a browser build, a downloadable, or both.</p>

      {/* Browser */}
      <Toggle icon={Globe} checked={form.hasBrowserBuild} onChange={(v) => update({ hasBrowserBuild: v })} label="Play in browser" description="HTML5 build that runs in the catalog — no install." />
      {form.hasBrowserBuild && (
        <div className="ml-1 space-y-3 border-l-2 border-ember-500/30 pl-4">
          <Field label="Hosted build URL" hint="Or upload an HTML/zip build below.">
            <input className="input" value={form.browserBuildUrl ?? ''} onChange={(e) => update({ browserBuildUrl: e.target.value || null })} placeholder="https://…/index.html" />
          </Field>
          <input ref={htmlRef} type="file" accept=".html,.zip,.htm" hidden onChange={(e) => pickHtml(e.target.files?.[0])} />
          <button onClick={() => htmlRef.current?.click()} className="btn btn-ghost">
            <Upload className="h-4 w-4" /> Upload HTML build
          </button>
          {form.browserBuildFile && <p className="text-xs text-emerald-400">{form.browserBuildFile.name} attached ✓</p>}
        </div>
      )}

      {/* Download */}
      <Toggle icon={Download} checked={form.hasDownloadBuild} onChange={(v) => update({ hasDownloadBuild: v })} label="Downloadable build" description="A .exe / .zip players download and install." />
      {form.hasDownloadBuild && (
        <div className="ml-1 space-y-3 border-l-2 border-ember-500/30 pl-4">
          <input ref={dlRef} type="file" accept=".zip,.exe,.dmg,.AppImage,.rar,.7z" hidden onChange={(e) => pickDownload(e.target.files?.[0])} />
          <button onClick={() => dlRef.current?.click()} className="btn btn-ghost">
            <Upload className="h-4 w-4" /> Upload build file
          </button>
          {form.downloadFile && (
            <p className="text-sm text-mist-300">
              <span className="font-mono text-emerald-400">{form.downloadFile.name}</span>{' '}
              <span className="text-mist-500">· {bytesToMB(form.downloadFile.size)} MB</span>
            </p>
          )}
          <Field label="Platforms">
            <div className="flex flex-wrap gap-2">
              {PLATFORMS.map((p) => {
                const on = form.downloadPlatforms.includes(p);
                return (
                  <button
                    key={p}
                    onClick={() =>
                      update({ downloadPlatforms: on ? form.downloadPlatforms.filter((x) => x !== p) : [...form.downloadPlatforms, p] })
                    }
                    className={`cursor-pointer rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                      on ? 'border-ember-500/50 bg-ember-500/10 text-ember-400' : 'border-iron-600 text-mist-300 hover:border-iron-500'
                    }`}
                  >
                    {p}
                  </button>
                );
              })}
            </div>
          </Field>
        </div>
      )}

      <Toggle icon={Users2} checked={form.supportsMultiplayer} onChange={(v) => update({ supportsMultiplayer: v })} label="Browser multiplayer" description="Flag this game as supporting real-time multiplayer in the browser." />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 3 — Pricing
// ---------------------------------------------------------------------------
function PricingStep({ form, update }: { form: CreateGameInput; update: (p: Partial<CreateGameInput>) => void }) {
  const setBenefit = (i: number, v: string) => {
    const benefits = [...form.subscription.benefits];
    benefits[i] = v;
    update({ subscription: { ...form.subscription, benefits } });
  };
  return (
    <div className="space-y-6">
      <h2 className="text-lg font-700 text-mist-50">Pricing & monetization</h2>

      <div className="grid grid-cols-2 gap-3">
        {(['free', 'paid'] as const).map((m) => (
          <button
            key={m}
            onClick={() => update({ pricingModel: m })}
            className={`cursor-pointer rounded-xl border p-4 text-left transition-colors ${
              form.pricingModel === m ? 'border-ember-500/60 bg-ember-500/5' : 'border-iron-700 hover:border-iron-600'
            }`}
          >
            <p className="font-600 text-mist-100">{m === 'free' ? 'Free' : 'Paid (buy once)'}</p>
            <p className="text-xs text-mist-400">{m === 'free' ? 'Anyone can add it to their library.' : 'Players pay once and own it forever.'}</p>
          </button>
        ))}
      </div>

      {form.pricingModel === 'paid' && (
        <div className="grid gap-5 sm:grid-cols-2">
          <Field label="Price (₽)">
            <input type="number" min={0} className="input font-mono" value={form.price || ''} onChange={(e) => update({ price: Number(e.target.value) })} placeholder="499" />
          </Field>
          <Field label="Friend Pack discount (%)" hint="Discount when an owner gifts a copy to a friend.">
            <div className="flex items-center gap-3">
              <Gift className="h-5 w-5 text-rose-400" />
              <input type="range" min={0} max={75} step={5} value={form.friendPackDiscount} onChange={(e) => update({ friendPackDiscount: Number(e.target.value) })} className="flex-1 accent-rose-500" />
              <span className="w-10 text-right font-mono text-sm text-rose-400">{form.friendPackDiscount}%</span>
            </div>
          </Field>
        </div>
      )}

      {/* Subscription */}
      <div className="border-t border-iron-700 pt-5">
        <Toggle icon={Repeat} checked={form.subscription.enabled} onChange={(v) => update({ subscription: { ...form.subscription, enabled: v } })} label="Offer an author subscription" description="Players back you monthly. You set the price and the perks." />
        {form.subscription.enabled && (
          <div className="ml-1 mt-4 space-y-4 border-l-2 border-ember-500/30 pl-4">
            <Field label="Monthly price (₽)">
              <input type="number" min={1} className="input w-40 font-mono" value={form.subscription.price || ''} onChange={(e) => update({ subscription: { ...form.subscription, price: Number(e.target.value) } })} placeholder="149" />
            </Field>
            <Field label="Subscriber benefits" hint="What backers get. One per line.">
              <div className="space-y-2">
                {form.subscription.benefits.map((b, i) => (
                  <div key={i} className="flex gap-2">
                    <input className="input" value={b} onChange={(e) => setBenefit(i, e.target.value)} placeholder="e.g. Early access builds" />
                    {form.subscription.benefits.length > 1 && (
                      <button onClick={() => update({ subscription: { ...form.subscription, benefits: form.subscription.benefits.filter((_, x) => x !== i) } })} className="btn btn-ghost px-2.5" aria-label="Remove benefit">
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                ))}
                <button onClick={() => update({ subscription: { ...form.subscription, benefits: [...form.subscription.benefits, ''] } })} className="inline-flex cursor-pointer items-center gap-1 text-sm text-ember-400 hover:text-ember-500">
                  <Plus className="h-4 w-4" /> Add benefit
                </button>
              </div>
            </Field>
            <Field label="Author chat link (perk)" hint="Shown only to active subscribers — e.g. a private Discord invite.">
              <input className="input" value={form.subscription.chatLink ?? ''} onChange={(e) => update({ subscription: { ...form.subscription, chatLink: e.target.value } })} placeholder="https://discord.gg/…" />
            </Field>
          </div>
        )}
      </div>

      {/* Demo Day */}
      <div className="border-t border-iron-700 pt-5">
        <Toggle icon={CalendarClock} checked={form.demoDay.enabled} onChange={(v) => update({ demoDay: { ...form.demoDay, enabled: v } })} label="Schedule a Demo Day" description="A free-to-play window, like Steam free weekends." />
        {form.demoDay.enabled && (
          <div className="ml-1 mt-4 grid gap-4 border-l-2 border-ember-500/30 pl-4 sm:grid-cols-2">
            <Field label="Starts">
              <input type="datetime-local" className="input" value={toLocalInput(form.demoDay.startsAt)} onChange={(e) => update({ demoDay: { ...form.demoDay, startsAt: fromLocalInput(e.target.value) } })} />
            </Field>
            <Field label="Ends">
              <input type="datetime-local" className="input" value={toLocalInput(form.demoDay.endsAt)} onChange={(e) => update({ demoDay: { ...form.demoDay, endsAt: fromLocalInput(e.target.value) } })} />
            </Field>
          </div>
        )}
      </div>
    </div>
  );
}

function toLocalInput(iso: string | null): string {
  if (!iso) return '';
  const d = new Date(iso);
  const tz = d.getTimezoneOffset() * 60000;
  return new Date(d.getTime() - tz).toISOString().slice(0, 16);
}
function fromLocalInput(v: string): string | null {
  return v ? new Date(v).toISOString() : null;
}

// ---------------------------------------------------------------------------
// Step 4 — Page style (itch.io-like customisation)
// ---------------------------------------------------------------------------
function StyleStep({ form, update }: { form: CreateGameInput; update: (p: Partial<CreateGameInput>) => void }) {
  const bgRef = useRef<HTMLInputElement>(null);
  const setTheme = (patch: Partial<GameTheme>) => update({ theme: { ...form.theme, ...patch } });
  const bgPreview = useMemo(
    () => (form.backgroundFile ? URL.createObjectURL(form.backgroundFile) : null),
    [form.backgroundFile],
  );
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-ember-400" />
        <h2 className="text-lg font-700 text-mist-50">Customise your game page</h2>
      </div>
      <p className="-mt-3 text-sm text-mist-400">Give your page its own identity — this drives the cover art and accent colours.</p>

      <Field label="Accent presets">
        <div className="flex flex-wrap gap-2">
          {ACCENT_PRESETS.map((p) => {
            const active = form.theme.accent === p.accent && form.theme.accent2 === p.accent2;
            return (
              <button
                key={p.name}
                onClick={() => setTheme({ accent: p.accent, accent2: p.accent2, background: p.background })}
                className={`cursor-pointer rounded-xl border p-1.5 transition-transform hover:scale-105 ${active ? 'border-ember-500' : 'border-iron-700'}`}
                title={p.name}
              >
                <span className="block h-9 w-14 rounded-sm" style={{ backgroundColor: p.accent }} />
              </button>
            );
          })}
        </div>
      </Field>

      <div className="grid gap-5 sm:grid-cols-3">
        <Field label="Accent">
          <ColorInput value={form.theme.accent} onChange={(v) => setTheme({ accent: v })} />
        </Field>
        <Field label="Accent 2">
          <ColorInput value={form.theme.accent2} onChange={(v) => setTheme({ accent2: v })} />
        </Field>
        <Field label="Mat color">
          <ColorInput value={form.theme.background} onChange={(v) => setTheme({ background: v })} />
        </Field>
      </div>

      <Field label="Wallpaper" hint="Shown behind your page on the sides. Leave blank to use your cover.">
        <input ref={bgRef} type="file" accept="image/*" hidden onChange={(e) => update({ backgroundFile: e.target.files?.[0] ?? null })} />
        <div className="flex items-center gap-4">
          {bgPreview ? (
            <div className="h-16 w-28 overflow-hidden rounded-lg border border-iron-700">
              <img src={bgPreview} alt="Wallpaper preview" className="h-full w-full object-cover" />
            </div>
          ) : (
            <div className="flex h-16 w-28 items-center justify-center rounded-lg border border-dashed border-iron-600 text-xs text-mist-500">
              No wallpaper
            </div>
          )}
          <div className="flex flex-col gap-2">
            <button type="button" onClick={() => bgRef.current?.click()} className="btn btn-ghost">
              <ImageIcon className="h-4 w-4" /> {form.backgroundFile ? 'Replace wallpaper' : 'Upload wallpaper'}
            </button>
            {form.backgroundFile && (
              <button type="button" onClick={() => update({ backgroundFile: null })} className="text-xs text-mist-500 hover:text-rose-400">
                Remove
              </button>
            )}
          </div>
        </div>
      </Field>

      <Field label="Layout">
        <div className="grid grid-cols-2 gap-3">
          {(['immersive', 'classic'] as const).map((l) => (
            <button
              key={l}
              onClick={() => setTheme({ layout: l })}
              className={`cursor-pointer rounded-xl border p-3 text-left text-sm transition-colors ${form.theme.layout === l ? 'border-ember-500/60 bg-ember-500/5 text-mist-100' : 'border-iron-700 text-mist-300 hover:border-iron-600'}`}
            >
              <span className="font-600 capitalize">{l}</span>
              <p className="text-xs text-mist-400">{l === 'immersive' ? 'Full-bleed banner & glow.' : 'Compact, classic listing.'}</p>
            </button>
          ))}
        </div>
      </Field>
    </div>
  );
}

function ColorInput({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="flex items-center gap-2">
      <input type="color" value={value} onChange={(e) => onChange(e.target.value)} className="h-10 w-12 cursor-pointer rounded-lg border border-iron-600 bg-transparent" />
      <input value={value} onChange={(e) => onChange(e.target.value)} className="input font-mono uppercase" />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 5 — Media
// ---------------------------------------------------------------------------
function MediaStep({ form, update }: { form: CreateGameInput; update: (p: Partial<CreateGameInput>) => void }) {
  const coverRef = useRef<HTMLInputElement>(null);
  const shotRef = useRef<HTMLInputElement>(null);
  const toast = useToast();

  const coverPreview = useMemo(() => (form.coverFile ? URL.createObjectURL(form.coverFile) : null), [form.coverFile]);
  const shotPreviews = useMemo(() => form.screenshotFiles.map((f) => URL.createObjectURL(f)), [form.screenshotFiles]);

  const onCover = (file?: File) => {
    if (!file) return;
    update({ coverFile: file });
  };
  const onShots = (files: FileList | null) => {
    if (!files) return;
    const arr = Array.from(files).slice(0, 6);
    update({ screenshotFiles: [...form.screenshotFiles, ...arr].slice(0, 8) });
    toast(`${arr.length} screenshot(s) added`, 'success');
  };

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-700 text-mist-50">Media</h2>
      <p className="-mt-3 text-sm text-mist-400">Optional — if you skip the cover, we generate on-theme key art for you.</p>

      <Field label="Cover image" hint="16:10 looks best. Shown on cards.">
        <input ref={coverRef} type="file" accept="image/*" hidden onChange={(e) => onCover(e.target.files?.[0])} />
        <div className="flex items-center gap-4">
          <div className="h-24 w-40 overflow-hidden rounded-xl border border-iron-700">
            <CoverArt game={{ title: form.title || 'Your game', coverImage: coverPreview, theme: form.theme, genre: form.genre }} showTitle={!coverPreview} />
          </div>
          <div className="flex flex-col gap-2">
            <button onClick={() => coverRef.current?.click()} className="btn btn-ghost">
              <ImageIcon className="h-4 w-4" /> {form.coverFile ? 'Replace cover' : 'Upload cover'}
            </button>
            {form.coverFile && (
              <button onClick={() => update({ coverFile: null })} className="text-xs text-mist-500 hover:text-rose-400">
                Remove cover
              </button>
            )}
          </div>
        </div>
      </Field>

      <Field label="Screenshots" hint="Up to 8. Shown in the gallery on your game page.">
        <input ref={shotRef} type="file" accept="image/*" multiple hidden onChange={(e) => onShots(e.target.files)} />
        <button onClick={() => shotRef.current?.click()} className="btn btn-ghost">
          <Upload className="h-4 w-4" /> Add screenshots
        </button>
        {shotPreviews.length > 0 && (
          <div className="mt-3 grid grid-cols-3 gap-2 sm:grid-cols-4">
            {shotPreviews.map((s, i) => (
              <div key={i} className="group relative aspect-video overflow-hidden rounded-lg border border-iron-700">
                <img src={s} alt={`Screenshot ${i + 1}`} className="h-full w-full object-cover" />
                <button onClick={() => update({ screenshotFiles: form.screenshotFiles.filter((_, x) => x !== i) })} className="absolute right-1 top-1 cursor-pointer rounded-md bg-iron-950/80 p-1 text-mist-200 opacity-0 transition-opacity group-hover:opacity-100" aria-label="Remove screenshot">
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>
        )}
      </Field>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 6 — Review
// ---------------------------------------------------------------------------
function ReviewStep({ form }: { form: CreateGameInput }) {
  const row = (label: string, value: React.ReactNode) => (
    <div className="flex items-start justify-between gap-4 border-b border-iron-800 py-2.5 last:border-0">
      <span className="text-sm text-mist-400">{label}</span>
      <span className="text-right text-sm font-500 text-mist-100">{value}</span>
    </div>
  );
  return (
    <div className="space-y-5">
      <h2 className="text-lg font-700 text-mist-50">Review & publish</h2>
      <div className="rounded-xl border border-iron-700 p-4">
        {row('Title', form.title || '—')}
        {row('Genre', form.genre)}
        {row('Builds', [form.hasBrowserBuild && 'Browser', form.hasDownloadBuild && 'Download'].filter(Boolean).join(' + ') || '—')}
        {row('Multiplayer', form.supportsMultiplayer ? 'Yes' : 'No')}
        {row('Price', form.pricingModel === 'free' ? 'Free' : RUB(form.price))}
        {form.pricingModel === 'paid' && form.friendPackDiscount > 0 && row('Friend Pack', `−${form.friendPackDiscount}%`)}
        {form.subscription.enabled && row('Subscription', `${RUB(form.subscription.price)}/mo · ${form.subscription.benefits.filter(Boolean).length} perks`)}
        {form.demoDay.enabled && row('Demo Day', 'Scheduled')}
        {row('Screenshots', String(form.screenshotFiles.length))}
      </div>
      <p className="text-sm text-mist-400">Looks good? Publishing makes your game live in the catalog immediately.</p>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Live preview card
// ---------------------------------------------------------------------------
function PreviewCard({ form }: { form: CreateGameInput }) {
  const cover = useMemo(() => (form.coverFile ? URL.createObjectURL(form.coverFile) : null), [form.coverFile]);
  const game = {
    title: form.title || 'Your game',
    coverImage: cover,
    theme: form.theme,
    genre: form.genre,
  };
  return (
    <div className="card overflow-hidden">
      <div className="aspect-[16/10]">
        <CoverArt game={game} showTitle={false} />
      </div>
      <div className="p-3.5">
        <div className="flex items-center justify-between gap-2">
          <h3 className="line-clamp-1 font-display font-600 text-mist-50">{game.title}</h3>
          <span className="font-mono text-sm font-600 text-ember-400">
            {form.pricingModel === 'free' ? 'Free' : RUB(form.price)}
          </span>
        </div>
        <p className="line-clamp-2 text-xs text-mist-400">{form.tagline || 'Your tagline appears here.'}</p>
        <div className="mt-2 flex items-center gap-2 text-mist-500">
          {form.hasBrowserBuild && <Globe className="h-3.5 w-3.5" />}
          {form.hasDownloadBuild && <Download className="h-3.5 w-3.5" />}
          {form.supportsMultiplayer && <Users2 className="h-3.5 w-3.5" />}
          {form.subscription.enabled && <Repeat className="h-3.5 w-3.5" />}
        </div>
      </div>
    </div>
  );
}
