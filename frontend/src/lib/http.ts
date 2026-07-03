// Thin fetch wrapper: base URL, bearer token, JSON, and uniform ApiError.
import { ApiError } from './errors';

const BASE = (import.meta.env.VITE_API_URL as string | undefined) || 'http://localhost:8080/api';
const TOKEN_KEY = 'indieforge_token';

function authHeader(): Record<string, string> {
  const t = localStorage.getItem(TOKEN_KEY);
  return t ? { Authorization: `Bearer ${t}` } : {};
}

async function request<T>(
  method: string,
  path: string,
  opts: { body?: unknown; form?: FormData } = {},
): Promise<T> {
  const headers: Record<string, string> = { ...authHeader() };
  let body: BodyInit | undefined;

  if (opts.form) {
    body = opts.form; // let the browser set the multipart boundary
  } else if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(opts.body);
  }

  let res: Response;
  try {
    res = await fetch(BASE + path, { method, headers, body });
  } catch {
    throw new ApiError(0, 'Network error — is the API running?');
  }

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    throw new ApiError(res.status, (data && data.error) || res.statusText);
  }
  return data as T;
}

export const http = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, { body }),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, { body }),
  del: <T>(path: string) => request<T>('DELETE', path),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, { body }),
  postForm: <T>(path: string, form: FormData) => request<T>('POST', path, { form }),
};
