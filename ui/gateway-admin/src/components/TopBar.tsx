import { useAdminScope } from "../app/AdminScopeContext";

type Props = {
  title: string;
  subtitle: string;
};

export function TopBar({ title, subtitle }: Props) {
  const { users, usersLoading, selectedUserEmail, setSelectedUserEmail } = useAdminScope();

  return (
    <header className="admin-surface flex items-center justify-between gap-4 px-6 py-4">
      <div>
        <div className="text-xs uppercase tracking-[0.2em] text-slate-400">Local Control Plane</div>
        <div className="mt-1 text-lg font-semibold text-slate-950">{title}</div>
      </div>
      <div className="flex items-center gap-3">
        <label className="text-xs text-slate-500">
          当前用户
          <select
            className="admin-input ml-3 min-w-56"
            value={selectedUserEmail}
            onChange={(event) => setSelectedUserEmail(event.target.value)}
          >
            <option value="">全部用户 / All Users</option>
            {users.map((user) => (
              <option key={user.user_email} value={user.user_email}>
                {user.label} ({user.user_email})
              </option>
            ))}
          </select>
        </label>
        <div className="admin-pill px-3 py-1.5 text-xs text-slate-600">
          {usersLoading ? "加载用户中" : subtitle}
        </div>
      </div>
    </header>
  );
}
