// Bulk image import. The queue lives here rather than in a page so an import
// keeps going while you browse the rest of the app.
import { api } from './api';
import { plural } from './format';
import { emitPin, toast } from './session.svelte';

const TYPES = new Set(['image/jpeg', 'image/png', 'image/gif', 'image/webp']);
// each upload holds a decoded image in the server's memory, so keep this small
const CONCURRENCY = 3;

type Job = { file: File; boardId: string };

export const imports = $state<{
  total: number;
  done: number;
  failed: (Job & { error: string })[];
}>({ total: 0, done: 0, failed: [] });

let queue: Job[] = [];
let running = 0;

export function importFiles(boardId: string, files: File[]) {
  const images = files.filter((f) => TYPES.has(f.type));
  const skipped = files.length - images.length;
  if (skipped > 0) toast(`Skipped ${plural(skipped, 'file')} that ${skipped === 1 ? 'isn’t' : 'aren’t'} JPEG, PNG, GIF or WebP`, 'error');
  add(images.map((file) => ({ file, boardId })));
}

export function retryFailed() {
  const jobs = imports.failed.map(({ file, boardId }) => ({ file, boardId }));
  imports.failed = [];
  imports.total -= jobs.length;
  add(jobs);
}

// Drops whatever hasn't started; uploads already in flight still finish.
export function cancelImport() {
  imports.total -= queue.length;
  queue = [];
  finish();
}

export function dismissImport() {
  imports.failed = [];
  imports.total = imports.done = 0;
}

function add(jobs: Job[]) {
  queue.push(...jobs);
  imports.total += jobs.length;
  pump();
}

function pump() {
  while (running < CONCURRENCY && queue.length > 0) upload(queue.shift()!);
}

async function upload(job: Job) {
  running++;
  const form = new FormData();
  form.append('board_id', job.boardId);
  form.append('image', job.file);
  try {
    emitPin({ type: 'created', pin: await api.createPin(form) });
    imports.done++;
  } catch (e) {
    imports.failed.push({ ...job, error: (e as Error).message });
  }
  running--;
  pump();
  finish();
}

function finish() {
  if (running > 0 || queue.length > 0 || imports.failed.length > 0) return;
  if (imports.done > 0) toast(`Imported ${plural(imports.done, 'pin')}`);
  dismissImport();
}

addEventListener('beforeunload', (e) => {
  if (running > 0 || queue.length > 0) e.preventDefault();
});

// --- reading drops ---

// Every file in a drop, including the contents of dropped folders.
export async function droppedFiles(dt: DataTransfer): Promise<File[]> {
  // the DataTransfer is emptied once this event ends, so grab entries before awaiting
  const entries = [...dt.items].map((i) => i.webkitGetAsEntry()).filter((e) => e !== null);
  if (entries.length === 0) return [...dt.files];
  return (await Promise.all(entries.map(walk))).flat();
}

async function walk(entry: FileSystemEntry): Promise<File[]> {
  if (entry.name.startsWith('.')) return []; // .DS_Store, and macOS ._photo.jpg sidecars
  try {
    if (entry.isFile) {
      return [await new Promise<File>((ok, fail) => (entry as FileSystemFileEntry).file(ok, fail))];
    }
    const reader = (entry as FileSystemDirectoryEntry).createReader();
    const found: File[] = [];
    // readEntries hands back ~100 at a time, and an empty batch at the end
    for (;;) {
      const batch = await new Promise<FileSystemEntry[]>((ok, fail) => reader.readEntries(ok, fail));
      if (batch.length === 0) return found;
      found.push(...(await Promise.all(batch.map(walk))).flat());
    }
  } catch {
    return [];
  }
}
