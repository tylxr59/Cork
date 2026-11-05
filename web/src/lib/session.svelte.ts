import { api } from './api';
import type { Board, Pin, User } from './types';

export const session = $state<{
  ready: boolean;
  user: User | null;
  needsSetup: boolean;
  registrationOpen: boolean;
}>({ ready: false, user: null, needsSetup: false, registrationOpen: false });

export async function refreshSession() {
  const s = await api.status();
  session.user = s.user;
  session.needsSetup = s.needs_setup;
  session.registrationOpen = s.registration_open;
  session.ready = true;
}

// --- cross-page notifications, so grids under a pin modal stay in sync ---

type PinEvent = { type: 'updated'; pin: Pin } | { type: 'deleted'; id: string } | { type: 'created'; pin: Pin };
const pinListeners = new Set<(e: PinEvent) => void>();

export function onPinEvent(fn: (e: PinEvent) => void) {
  pinListeners.add(fn);
  return () => pinListeners.delete(fn);
}

export function emitPin(e: PinEvent) {
  pinListeners.forEach((fn) => fn(e));
}

// --- the signed-in user's boards (used by pickers) ---

export const myBoards = $state<{ list: Board[]; loaded: boolean }>({ list: [], loaded: false });

export async function loadBoards(force = false) {
  if (myBoards.loaded && !force) return myBoards.list;
  myBoards.list = await api.boards();
  myBoards.loaded = true;
  return myBoards.list;
}

// --- toasts ---

export const toasts = $state<{ list: { id: number; text: string; kind: 'info' | 'error' }[] }>({ list: [] });
let toastId = 0;

export function toast(text: string, kind: 'info' | 'error' = 'info') {
  const id = ++toastId;
  toasts.list.push({ id, text, kind });
  setTimeout(() => {
    toasts.list = toasts.list.filter((t) => t.id !== id);
  }, 4000);
}

export function toastError(e: unknown) {
  toast(e instanceof Error ? e.message : String(e), 'error');
}
