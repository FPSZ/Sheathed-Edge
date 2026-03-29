type Props = {
  title: string;
  description: string;
  content: string;
};

export function DetailsDrawer({ title, description, content }: Props) {
  return (
    <aside className="hidden h-full w-80 bg-[var(--panel-muted)] shadow-[inset_1px_0_0_rgba(255,255,255,0.55)] xl:flex xl:flex-col">
      <div className="px-5 py-4">
        <div className="text-sm font-semibold text-slate-950">{title}</div>
        <p className="mt-1 text-xs leading-6 text-slate-500">{description}</p>
        <div className="admin-divider mt-4" />
      </div>
      <div className="flex-1 overflow-auto px-5 py-4">
        <pre className="admin-surface whitespace-pre-wrap break-words rounded-3xl p-4 text-xs leading-6 text-slate-700">
          {content}
        </pre>
      </div>
    </aside>
  );
}
