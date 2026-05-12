// Tiny "shadcn-style" component primitives. Plain Tailwind classes, no runtime,
// no class-variance-authority — keeps the surface area readable.

import { ButtonHTMLAttributes, HTMLAttributes, InputHTMLAttributes, TextareaHTMLAttributes, forwardRef } from "react";
import { cn } from "@/lib/utils";

type Variant = "primary" | "ghost" | "muted" | "danger";

const variants: Record<Variant, string> = {
  primary: "bg-accent text-bg hover:brightness-110",
  ghost: "bg-transparent text-accent border border-border hover:bg-accent/10",
  muted: "bg-panel text-muted border border-border hover:bg-hover",
  danger: "bg-err text-white hover:brightness-110",
};

export const Button = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }>(
  ({ className, variant = "primary", ...props }, ref) => (
    <button
      ref={ref}
      className={cn(
        "inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-[13px] font-medium transition disabled:opacity-50 disabled:cursor-not-allowed",
        variants[variant],
        className
      )}
      {...props}
    />
  )
);

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        "w-full rounded-md border border-border bg-panel px-3 py-2 text-[13px] text-fg outline-none focus:border-accent",
        className
      )}
      {...props}
    />
  )
);

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        "w-full min-h-[200px] rounded-md border border-border bg-panel px-3 py-2 font-mono text-[12px] text-fg outline-none focus:border-accent",
        className
      )}
      {...props}
    />
  )
);

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("rounded-lg border border-border bg-panel p-4", className)} {...props} />;
}

export function CardTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3
      className={cn("mb-3 text-[11px] font-medium uppercase tracking-[0.06em] text-muted", className)}
      {...props}
    />
  );
}

type PillVariant = "ok" | "warn" | "muted" | "add" | "del";
const pillVariants: Record<PillVariant, string> = {
  ok: "bg-ok/15 text-ok",
  warn: "bg-warm/15 text-warm",
  muted: "bg-muted/15 text-muted",
  add: "bg-add/15 text-add",
  del: "bg-del/15 text-del",
};

export function Pill({ variant = "muted", className, ...props }: HTMLAttributes<HTMLSpanElement> & { variant?: PillVariant }) {
  return (
    <span
      className={cn(
        "inline-block rounded-full px-2 py-0.5 text-[10px] font-medium",
        pillVariants[variant],
        className
      )}
      {...props}
    />
  );
}

export function KV({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <>
      <div className="text-muted">{k}</div>
      <div className="font-mono text-[12px] break-all">{v}</div>
    </>
  );
}

export function KVGrid({ children }: { children: React.ReactNode }) {
  return <div className="grid grid-cols-[130px_1fr] gap-x-3.5 gap-y-1.5">{children}</div>;
}

export function Crumb({ children }: { children: React.ReactNode }) {
  return <div className="text-[12px] text-muted">{children}</div>;
}

export function PageHeader({
  crumb,
  title,
  subtitle,
  right,
}: {
  crumb?: React.ReactNode;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  right?: React.ReactNode;
}) {
  return (
    <header className="mb-5 flex items-start justify-between gap-4">
      <div>
        {crumb && <Crumb>{crumb}</Crumb>}
        <h2 className="mt-0.5 text-[17px] font-semibold leading-tight">{title}</h2>
        {subtitle && <p className="mt-1.5 text-[12px] text-muted">{subtitle}</p>}
      </div>
      {right && <div className="flex items-center gap-2 flex-wrap">{right}</div>}
    </header>
  );
}
