import { useAdminScope } from "../app/AdminScopeContext";

type Props = {
  title: string;
  subtitle: string;
};

export function TopBar({ title, subtitle }: Props) {
  const { users, usersLoading, selectedUserEmail, setSelectedUserEmail } = useAdminScope();

  return (
    <header className="admin-surface flex items-center justify-between gap-6 px-6 py-4">
      <div className="min-w-0">
        <div className="text-xs uppercase tracking-[0.2em] text-slate-400">Local Control Plane</div>
        <div className="mt-1 flex min-w-0 items-baseline gap-3">
          <div className="shrink-0 text-lg font-semibold text-slate-950">{title}</div>
          <div className="truncate text-sm text-slate-400">{subtitle}</div>
        </div>
      </div>

      <div className="min-w-0">
        <select
          className="admin-input min-w-64 max-w-[22rem]"
          value={selectedUserEmail}
          onChange={(event) => setSelectedUserEmail(event.target.value)}
        >
          <option value="">{usersLoading ? "加载用户中..." : "全部用户 / All Users"}</option>
          {users.map((user) => (
            <option key={user.user_email} value={user.user_email}>
              {user.label} ({user.user_email})
            </option>
          ))}
        </select>
      </div>
    </header>
  );
}
