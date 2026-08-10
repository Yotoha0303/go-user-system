import {
  KeyRound,
  LogIn,
  LogOut,
  Menu,
  ShieldCheck,
  UserPlus,
  UserRound,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";
import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { logout } from "../../api/auth.api";
import { PermissionCode } from "../../api/types";
import { selectAuth, sessionCleared } from "../../app/authSlice";
import { useAppDispatch, useAppSelector } from "../../app/hooks";
import BrandMark from "../brand/brand-mark";
import Button from "../elements/button";
import { buttonClassName } from "../elements/button-styles";

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  `group flex h-10 items-center gap-3 rounded-md px-3 text-sm font-semibold transition-colors ${
    isActive
      ? "bg-white text-slate-950 shadow-sm"
      : "text-slate-300 hover:bg-white/10 hover:text-white"
  }`;

const Navbar = () => {
  const auth = useAppSelector(selectAuth);
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);
  const canManageAccess = auth.permissionCodes.includes(
    PermissionCode.adminRolesRead
  );

  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    if (!mobileOpen) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setMobileOpen(false);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [mobileOpen]);

  const handleLogout = async () => {
    try {
      await logout(auth.accessToken);
    } catch {
      // Local session cleanup must not depend on network availability.
    } finally {
      dispatch(sessionCleared());
      navigate("/auth/login", { replace: true });
    }
  };

  if (auth.status !== "authenticated") {
    return (
      <header className="relative z-20 border-b border-slate-200 bg-white">
        <nav className="mx-auto flex h-16 w-full max-w-7xl items-center gap-3 px-4 sm:px-6 lg:px-8">
          <NavLink to="/" className="mr-auto min-w-0" aria-label="Go User System home">
            <BrandMark />
          </NavLink>
          <NavLink
            to="/auth/login"
            className={buttonClassName({
              variant: "ghost",
              size: "sm",
              className: "px-2.5 sm:px-3",
            })}
          >
            <LogIn className="h-4 w-4" aria-hidden="true" />
            <span className="hidden sm:inline">Sign in</span>
          </NavLink>
          <NavLink
            to="/auth/signup"
            className={buttonClassName({ size: "sm" })}
          >
            <UserPlus className="h-4 w-4" aria-hidden="true" />
            Sign up
          </NavLink>
        </nav>
      </header>
    );
  }

  const displayName = auth.user?.nickname || auth.user?.username || "User";
  const roleLabel = auth.roleCodes[0]?.replaceAll("_", " ") || "Member";

  return (
    <>
      <header className="sticky top-0 z-30 flex h-16 items-center border-b border-slate-200 bg-white px-4 lg:hidden">
        <NavLink to="/profile" className="mr-auto min-w-0" aria-label="Go User System home">
          <BrandMark />
        </NavLink>
        <button
          type="button"
          className="grid h-10 w-10 place-items-center rounded-md border border-slate-200 text-slate-700 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-600"
          aria-label="Open navigation"
          title="Open navigation"
          aria-expanded={mobileOpen}
          aria-controls="primary-sidebar"
          onClick={() => setMobileOpen(true)}
        >
          <Menu className="h-5 w-5" aria-hidden="true" />
        </button>
      </header>

      {mobileOpen ? (
        <button
          type="button"
          className="fixed inset-0 z-40 bg-slate-950/40 lg:hidden"
          aria-label="Close navigation"
          onClick={() => setMobileOpen(false)}
        />
      ) : null}

      <aside
        id="primary-sidebar"
        className={`fixed inset-y-0 left-0 z-50 flex w-72 flex-col overflow-y-auto overscroll-contain border-r border-white/10 bg-[#162024] px-4 py-5 transition-transform duration-200 lg:visible lg:w-64 lg:translate-x-0 ${
          mobileOpen
            ? "visible translate-x-0"
            : "invisible -translate-x-full"
        }`}
      >
        <div className="flex h-10 shrink-0 items-center px-2">
          <NavLink to="/profile" className="mr-auto min-w-0" aria-label="Go User System home">
            <BrandMark inverse />
          </NavLink>
          <button
            type="button"
            className="grid h-9 w-9 place-items-center rounded-md text-slate-300 hover:bg-white/10 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-400 lg:hidden"
            aria-label="Close navigation"
            title="Close navigation"
            onClick={() => setMobileOpen(false)}
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        <div className="mt-8 shrink-0 px-3 text-xs font-bold uppercase text-slate-500">
          Workspace
        </div>
        <nav
          aria-label="Primary navigation"
          className="mt-3 shrink-0 space-y-1"
        >
          <NavLink to="/profile" end className={navLinkClass}>
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
              Access control
            </NavLink>
          ) : null}
        </nav>

        <div className="mt-auto shrink-0 border-t border-white/10 pt-4">
          <div className="mb-3 flex min-w-0 items-center gap-3 px-2">
            <span
              className="grid h-9 w-9 shrink-0 place-items-center rounded-md bg-teal-400 text-sm font-bold text-slate-950"
              aria-hidden="true"
            >
              {displayName.charAt(0).toUpperCase()}
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold text-white">{displayName}</p>
              <p className="truncate text-xs capitalize text-slate-400">{roleLabel}</p>
            </div>
          </div>
          <Button
            className="w-full justify-start text-slate-300 hover:bg-white/10 hover:text-white"
            type="button"
            variant="ghost"
            size="sm"
            icon={<LogOut className="h-4 w-4" aria-hidden="true" />}
            onClick={() => void handleLogout()}
          >
            Sign out
          </Button>
        </div>
      </aside>
    </>
  );
};

export default Navbar;
