import { useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { CheckCircle2, XCircle } from 'lucide-react';
import type { Game, Payment } from '@/lib/types';
import { api } from '@/lib/api';
import { PageLoader, Spinner } from '@/components/ui';
import { RUB } from '@/lib/constants';

/**
 * Landing page YooKassa redirects back to. Polls the payment until the webhook
 * has marked it succeeded (or it was canceled).
 */
export function ReturnPage() {
  const [params] = useSearchParams();
  const paymentId = params.get('paymentId') || '';
  const navigate = useNavigate();
  const [payment, setPayment] = useState<Payment | null>(null);
  const [game, setGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);
  const tries = useRef(0);

  useEffect(() => {
    if (!paymentId) {
      setLoading(false);
      return;
    }
    let active = true;
    let timer: number;

    const poll = async () => {
      try {
        const { payment, game } = await api.getPayment(paymentId);
        if (!active) return;
        setPayment(payment);
        setGame(game);
        setLoading(false);
        tries.current += 1;
        if (payment.status === 'pending' && tries.current < 20) {
          timer = window.setTimeout(poll, 2000);
        }
      } catch {
        if (active) setLoading(false);
      }
    };
    poll();
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [paymentId]);

  if (loading) return <PageLoader label="Confirming your payment…" />;
  if (!payment || !game) {
    return (
      <div className="container-page py-16 text-center text-mist-300">Payment not found.</div>
    );
  }

  const succeeded = payment.status === 'succeeded';
  const pending = payment.status === 'pending';

  return (
    <div className="container-page flex min-h-[calc(100vh-4rem)] items-center justify-center py-10">
      <div className="card w-full max-w-md p-8 text-center">
        {succeeded ? (
          <CheckCircle2 className="mx-auto h-14 w-14 text-emerald-400" />
        ) : pending ? (
          <Spinner className="mx-auto h-12 w-12 text-ember-500" />
        ) : (
          <XCircle className="mx-auto h-14 w-14 text-rose-400" />
        )}
        <h1 className="mt-4 text-xl font-700 text-mist-50">
          {succeeded ? 'Payment confirmed' : pending ? 'Waiting for confirmation…' : 'Payment not completed'}
        </h1>
        <p className="mt-1 text-sm text-mist-400">
          {succeeded
            ? payment.kind === 'subscription'
              ? `You're now subscribed to ${game.developerName}.`
              : payment.kind === 'friend-pack'
                ? `${game.title} was gifted to ${payment.friendUsername}.`
                : `${game.title} is now in your library.`
            : pending
              ? 'This can take a few seconds after you pay. The page updates automatically.'
              : 'No charge was made. You can try again from the game page.'}
        </p>
        <p className="mt-3 font-mono text-lg font-700 text-ember-400">{RUB(payment.amount)}</p>

        <div className="mt-6 flex flex-col gap-2">
          {succeeded && (payment.kind === 'purchase' || payment.kind === 'subscription') && (
            <button onClick={() => navigate('/library')} className="btn btn-primary btn-lg w-full">
              Go to library
            </button>
          )}
          <button onClick={() => navigate(`/game/${game.slug}`)} className="btn btn-ghost w-full">
            Back to {game.title}
          </button>
        </div>
      </div>
    </div>
  );
}
