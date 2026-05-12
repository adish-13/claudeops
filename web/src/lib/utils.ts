// Misc shared helpers.
import clsx, { ClassValue } from "clsx";

export function cn(...inputs: ClassValue[]) {
  return clsx(inputs);
}

export function humanAgo(iso: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  const now = Date.now();
  const sec = Math.max(0, Math.floor((now - then) / 1000));
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  if (day < 30) return `${day}d ago`;
  const mo = Math.floor(day / 30);
  return `${mo}mo ago`;
}

export function shortId(id: string, n = 8): string {
  return id.length <= n ? id : id.slice(0, n);
}

export function shortPath(p: string): string {
  // server already collapses ~/ for us in most places, but in case not:
  return p;
}
