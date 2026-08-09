import { Pencil } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { errorMessage } from "../../api/errors";
import { getCurrentUser } from "../../api/user.api";
import { profileUpdated, selectCurrentUser } from "../../app/authSlice";
import { useAppDispatch, useAppSelector } from "../../app/hooks";
import AccountTabs from "../../components/account/account-tabs";
import Alert from "../../components/elements/alert";
import { buttonClassName } from "../../components/elements/button-styles";
import Spinner from "../../components/elements/spinner";

const formatDateTime = (value?: string) => {
  if (!value) return "Never";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
};

const ProfilePage = () => {
  const profile = useAppSelector(selectCurrentUser);
  const dispatch = useAppDispatch();
  const location = useLocation();
  const notice = (location.state as { notice?: string } | null)?.notice;
  const [loading, setLoading] = useState(!profile);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const user = await getCurrentUser();
        if (active) dispatch(profileUpdated(user));
      } catch (error) {
        if (active) setLoadError(errorMessage(error, "Unable to load profile."));
      } finally {
        if (active) setLoading(false);
      }
    };
    void load();
    return () => {
      active = false;
    };
  }, [dispatch]);

  return (
    <section className="mx-auto w-full max-w-3xl">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm font-semibold text-blue-700">Account</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-950">Profile</h1>
          <p className="mt-2 text-sm text-slate-600">Your account identity and current status.</p>
        </div>
        <Link
          to="/profile/edit-nickname"
          className={buttonClassName({ variant: "secondary" })}
        >
          <Pencil className="h-4 w-4" aria-hidden="true" />
          Edit nickname
        </Link>
      </div>

      <AccountTabs />
      {notice ? <div className="mb-5"><Alert tone="success">{notice}</Alert></div> : null}
      {loadError ? <div className="mb-5"><Alert>{loadError}</Alert></div> : null}

      {loading ? (
        <div className="flex items-center gap-2 py-10 text-sm text-slate-600">
          <Spinner size="sm" /> Loading profile
        </div>
      ) : profile ? (
        <dl className="divide-y divide-slate-200 border-y border-slate-200 bg-white">
          {[
            ["User ID", String(profile.id)],
            ["Username", profile.username],
            ["Nickname", profile.nickname || "Not set"],
            ["Status", profile.status === 1 ? "Active" : "Disabled"],
            ["Last login", formatDateTime(profile.last_login_at)],
          ].map(([label, value]) => (
            <div key={label} className="grid gap-1 px-4 py-4 sm:grid-cols-[10rem_1fr] sm:px-5">
              <dt className="text-sm font-medium text-slate-500">{label}</dt>
              <dd className="break-words text-sm font-semibold text-slate-900">{value}</dd>
            </div>
          ))}
        </dl>
      ) : null}
    </section>
  );
};

export default ProfilePage;
