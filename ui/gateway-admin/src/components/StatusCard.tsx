import clsx from "clsx";

type Props = {
  title: string;
  status: string;
  subtitle?: string;
  meta?: string;
};

export function StatusCard({ title, status, subtitle, meta }: Props) {
  const tone =
    status === "ok"
      ? "border-emerald-200 bg-emerald-50 text-emerald-700"
      : "border-rose-200 bg-rose-50 text-rose-700";

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-slate-900">{title}</div>
          {subtitle ? <div className="mt-1 text-xs text-slate-500">{subtitle}</div> : null}
        </div>
        <span className={clsx("rounded-full border px-2.5 py-1 text-xs font-medium", tone)}>
          {status}
        </span>
      </div>
      {meta ? <div className="mt-3 text-xs text-slate-600">{meta}</div> : null}
    </div>
  );
}
