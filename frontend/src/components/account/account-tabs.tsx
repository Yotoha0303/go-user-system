import { KeyRound, UserRound } from "lucide-react";
import { NavLink } from "react-router-dom";

const tabClass = ({ isActive }: { isActive: boolean }) =>
  `inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm font-semibold transition-colors ${
    isActive
      ? "bg-white text-slate-950 shadow-sm ring-1 ring-slate-200"
      : "text-slate-500 hover:bg-white/70 hover:text-slate-900"
  }`;

const AccountTabs = () => (
  <nav
    aria-label="Account settings"
    className="mb-7 inline-flex rounded-lg border border-slate-200 bg-slate-100 p-1"
  >
    <NavLink to="/profile" end className={tabClass}>
      <UserRound className="h-4 w-4" aria-hidden="true" />
      Profile
    </NavLink>
    <NavLink to="/security/password" className={tabClass}>
      <KeyRound className="h-4 w-4" aria-hidden="true" />
      Security
    </NavLink>
  </nav>
);

export default AccountTabs;
