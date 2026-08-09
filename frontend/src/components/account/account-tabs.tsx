import { NavLink } from "react-router-dom";

const tabClass = ({ isActive }: { isActive: boolean }) =>
  `border-b-2 px-1 pb-3 text-sm font-semibold ${
    isActive
      ? "border-blue-600 text-blue-700"
      : "border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-800"
  }`;

const AccountTabs = () => (
  <nav aria-label="Account settings" className="mb-7 flex gap-6 border-b border-slate-200">
    <NavLink to="/profile" end className={tabClass}>
      Profile
    </NavLink>
    <NavLink to="/security/password" className={tabClass}>
      Security
    </NavLink>
  </nav>
);

export default AccountTabs;
