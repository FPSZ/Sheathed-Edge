import type { ReactNode } from "react";

import clsx from "clsx";

type Props = {
  title: string;
  status: string;
  subtitle?: string;
  meta?: string;
  actions?: ReactNode;
};

export function StatusCard({ title, status, subtitle, meta, actions }: Props) {
  const tone = status === "ok" ? "admin-badge ok" : "admin-badge down";

  return (
    <div className="admin-surface rounded-3xl p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-slate-900">{title}</div>
          {subtitle ? <div className="mt-1 text-xs text-slate-500">{subtitle}</div> : null}
        </div>
        <span className={clsx(tone)}>
          {status}
        </span>
      </div>
      {meta ? <div className="mt-3 text-xs text-slate-600">{meta}</div> : null}
      {actions ? <div className="mt-4 flex flex-wrap gap-2">{actions}</div> : null}
    </div>
  );
}
