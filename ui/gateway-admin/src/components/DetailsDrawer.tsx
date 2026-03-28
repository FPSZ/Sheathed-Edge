type Props = {
  title: string;
  description: string;
  content: string;
};

export function DetailsDrawer({ title, description, content }: Props) {
  return (
    <aside className="hidden h-full w-80 border-l border-slate-200 bg-[var(--panel-muted)] xl:flex xl:flex-col">
      <div className="border-b border-slate-200 px-5 py-4">
        <div className="text-sm font-semibold text-slate-950">{title}</div>
        <p className="mt-1 text-xs leading-6 text-slate-500">{description}</p>
      </div>
      <div className="flex-1 overflow-auto px-5 py-4">
        <pre className="whitespace-pre-wrap break-words rounded-2xl border border-slate-200 bg-white p-4 text-xs leading-6 text-slate-700 shadow-sm">
          {content}
        </pre>
      </div>
    </aside>
  );
}
