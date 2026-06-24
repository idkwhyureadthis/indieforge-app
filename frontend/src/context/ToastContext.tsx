import { createContext, useCallback, useContext, useState } from 'react';
import type { ReactNode } from 'react';
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react';

type ToastKind = 'success' | 'error' | 'info';
interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

interface ToastCtx {
  toast: (message: string, kind?: ToastKind) => void;
}

const Ctx = createContext<ToastCtx | null>(null);

const ICONS = { success: CheckCircle2, error: AlertCircle, info: Info };
const ACCENT = {
  success: 'text-emerald-400',
  error: 'text-rose-400',
  info: 'text-ember-400',
};

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const remove = useCallback((id: number) => {
    setToasts((t) => t.filter((x) => x.id !== id));
  }, []);

  const toast = useCallback(
    (message: string, kind: ToastKind = 'info') => {
      const id = Date.now() + Math.random();
      setToasts((t) => [...t, { id, kind, message }]);
      setTimeout(() => remove(id), 4200);
    },
    [remove],
  );

  return (
    <Ctx.Provider value={{ toast }}>
      {children}
      <div className="pointer-events-none fixed bottom-4 right-4 z-50 flex w-[min(92vw,22rem)] flex-col gap-2">
        {toasts.map((t) => {
          const Icon = ICONS[t.kind];
          return (
            <div
              key={t.id}
              role="status"
              className="pointer-events-auto flex items-start gap-3 rounded-xl border border-iron-600 bg-iron-800/95 px-4 py-3 shadow-2xl backdrop-blur animate-[slideIn_.2s_ease-out]"
            >
              <Icon className={`mt-0.5 h-5 w-5 shrink-0 ${ACCENT[t.kind]}`} />
              <p className="flex-1 text-sm text-mist-100">{t.message}</p>
              <button
                onClick={() => remove(t.id)}
                aria-label="Dismiss notification"
                className="cursor-pointer text-mist-500 transition-colors hover:text-mist-200"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          );
        })}
      </div>
      <style>{`@keyframes slideIn{from{opacity:0;transform:translateY(8px)}to{opacity:1;transform:translateY(0)}}`}</style>
    </Ctx.Provider>
  );
}

export function useToast() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx.toast;
}
