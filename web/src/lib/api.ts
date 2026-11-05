import type { AdminUser, Board, Pin, ScrapedPage, Status, User } from './types';

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'X-Cork': '1' };
  let payload: BodyInit | undefined;
  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    payload = JSON.stringify(body);
  }
  let res: Response;
  try {
    res = await fetch(path, { method, headers, body: payload, credentials: 'same-origin' });
  } catch {
    throw new ApiError('Could not reach the Cork server', 0);
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    if (res.status === 401 && onUnauthorized) onUnauthorized();
    throw new ApiError(data.error ?? `Request failed (${res.status})`, res.status);
  }
  return data as T;
}

let onUnauthorized: (() => void) | null = null;
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

const get = <T>(p: string) => request<T>('GET', p);
const post = <T>(p: string, b?: unknown) => request<T>('POST', p, b);
const put = <T>(p: string, b?: unknown) => request<T>('PUT', p, b);
const patch = <T>(p: string, b?: unknown) => request<T>('PATCH', p, b);
const del = <T>(p: string) => request<T>('DELETE', p);

type Ok = { ok: boolean };

export const api = {
  status: () => get<Status>('/api/status'),
  login: (username: string, password: string) => post<Ok>('/api/auth/login', { username, password }),
  register: (username: string, password: string) => post<User>('/api/auth/register', { username, password }),
  logout: () => post<Ok>('/api/auth/logout'),
  changePassword: (current_password: string, new_password: string) =>
    put<Ok>('/api/me/password', { current_password, new_password }),

  boards: () => get<Board[]>('/api/boards'),
  board: (id: string) => get<{ board: Board; pins: Pin[] }>(`/api/boards/${id}`),
  createBoard: (b: { name: string; description?: string; is_public?: boolean }) => post<Board>('/api/boards', b),
  updateBoard: (id: string, b: Partial<Pick<Board, 'name' | 'description' | 'is_public'>>) =>
    patch<Board>(`/api/boards/${id}`, b),
  deleteBoard: (id: string) => del<Ok>(`/api/boards/${id}`),
  orderBoards: (ids: string[]) => put<Ok>('/api/boards/order', { ids }),
  orderPins: (boardId: string, ids: string[]) => put<Ok>(`/api/boards/${boardId}/order`, { ids }),

  pins: (q: string, offset: number, limit = 60) =>
    get<{ pins: Pin[]; has_more: boolean }>(
      `/api/pins?${new URLSearchParams({ q, offset: String(offset), limit: String(limit) })}`,
    ),
  pin: (id: string) => get<Pin>(`/api/pins/${id}`),
  createPin: (form: FormData) => post<Pin>('/api/pins', form),
  updatePin: (id: string, p: Partial<Pick<Pin, 'title' | 'description' | 'source_url' | 'board_id' | 'price' | 'currency'>>) =>
    patch<Pin>(`/api/pins/${id}`, p),
  replacePinImage: (id: string, form: FormData) => put<Pin>(`/api/pins/${id}/image`, form),
  deletePin: (id: string) => del<Ok>(`/api/pins/${id}`),
  scrape: (url: string) => post<ScrapedPage>('/api/scrape', { url }),

  adminUsers: () => get<AdminUser[]>('/api/admin/users'),
  adminCreateUser: (u: { username: string; password: string; is_admin: boolean }) =>
    post<AdminUser>('/api/admin/users', u),
  adminUpdateUser: (id: string, u: { is_admin?: boolean; password?: string }) =>
    patch<Ok>(`/api/admin/users/${id}`, u),
  adminDeleteUser: (id: string) => del<Ok>(`/api/admin/users/${id}`),
  adminSettings: () => get<{ registration_open: boolean }>('/api/admin/settings'),
  adminSaveSettings: (s: { registration_open: boolean }) => put<{ registration_open: boolean }>('/api/admin/settings', s),
};
