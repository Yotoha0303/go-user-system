import { KeyRound, LogIn, LogOut, ShieldCheck, UserPlus, UserRound } from "lucide-react";
import { NavLink, useNavigate } from "react-router-dom";
import { logout } from "../../api/auth.api";
import { PermissionCode } from "../../api/types";
import { selectAuth, sessionCleared } from "../../app/authSlice";
import { useAppDispatch, useAppSelector } from "../../app/hooks";
import Button from "../elements/button";
import { buttonClassName } from "../elements/button-styles";

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  `inline-flex min-h-9 items-center justify-center gap-2 rounded-md px-3 text-sm font-semibold transition-colors sm:justify-start ${
    isActive
      ? "bg-slate-900 text-white"
      : "text-slate-600 hover:bg-slate-100 hover:text-slate-950"
  }`;

const Navbar = () => {
  const auth = useAppSelector(selectAuth);
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const canManageAccess = auth.permissionCodes.includes(
    PermissionCode.adminRolesRead
  );

  const handleLogout = async () => {
    try {
      await logout();
    } catch {
      // Local session cleanup must not depend on network availability.
    } finally {
      dispatch(sessionCleared());
      navigate("/auth/login", { replace: true });
    }
  };

  return (
    <header className="border-b border-slate-200 bg-white">
      <nav className="mx-auto flex min-h-16 w-full max-w-6xl flex-wrap items-center gap-3 px-4 py-3 sm:px-6 lg:px-8">
        <NavLink to="/" className="w-full text-base font-bold text-slate-950 sm:mr-auto sm:w-auto">
          Go User System
        </NavLink>

        {auth.status === "authenticated" ? (
          <div className="grid w-full grid-cols-2 gap-2 sm:flex sm:w-auto sm:items-center">
            <NavLink to="/profile" className={navLinkClass}>
              <UserRound className="h-4 w-4" aria-hidden="true" />
              Profile
            </NavLink>
            <NavLink to="/security/password" className={navLinkClass}>
              <KeyRound className="h-4 w-4" aria-hidden="true" />
              Security
            </NavLink>
            {canManageAccess ? (
              <NavLink to="/admin/access" className={navLinkClass}>
                <ShieldCheck className="h-4 w-4" aria-hidden="true" />
                Access
              </NavLink>
            ) : null}
            <Button
              className="w-full sm:w-auto"
              type="button"
              variant="ghost"
              size="sm"
              icon={<LogOut className="h-4 w-4" aria-hidden="true" />}
              onClick={() => void handleLogout()}
            >
              Sign out
            </Button>
          </div>
        ) : (
          <div className="flex w-full gap-2 sm:w-auto">
            <NavLink
              to="/auth/login"
              className={buttonClassName({
                variant: "ghost",
                size: "sm",
                className: "min-w-0 flex-1 sm:flex-none",
              })}
            >
              <LogIn className="h-4 w-4" aria-hidden="true" />
              Sign in
            </NavLink>
            <NavLink
              to="/auth/signup"
              className={buttonClassName({
                size: "sm",
                className: "min-w-0 flex-1 sm:flex-none",
              })}
            >
              <UserPlus className="h-4 w-4" aria-hidden="true" />
              Sign up
            </NavLink>
          </div>
        )}
      </nav>
    </header>
  );
};

export default Navbar;
