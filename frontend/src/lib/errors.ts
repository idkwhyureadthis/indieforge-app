// Shared error type so `instanceof ApiError` works across both the real HTTP
// client and the in-browser mock.
export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = 'ApiError';
  }
}
