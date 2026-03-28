type Props = {
  title: string;
  subtitle: string;
};

export function TopBar({ title, subtitle }: Props) {
  return (
    <header className="flex items-center justify-between border-b border-slate-200 bg-white px-6 py-4">
      <div>
        <div className="text-xs uppercase tracking-[0.2em] text-slate-400">Local Control Plane</div>
        <div className="mt-1 text-lg font-semibold text-slate-950">{title}</div>
      </div>
      <div className="rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs text-slate-600">
        {subtitle}
      </div>
    </header>
  );
}
