import { Pencil, UserRound } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { errorMessage } from "../../api/errors";
import { getCurrentUser } from "../../api/user.api";
import { profileUpdated, selectAuth } from "../../app/authSlice";
import { useAppDispatch, useAppSelector } from "../../app/hooks";
import AccountTabs from "../../components/account/account-tabs";
import Alert from "../../components/elements/alert";
import { buttonClassName } from "../../components/elements/button-styles";
import PageHeader from "../../components/elements/page-header";
import Spinner from "../../components/elements/spinner";

const formatDateTime = (value?: string) => {
  if (!value) return "Never";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
};

const ProfilePage = () => {
  const { roleCodes, user: profile } = useAppSelector(selectAuth);
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

  const displayName = profile?.nickname || profile?.username || "Account";
  const initial = displayName.charAt(0).toUpperCase();

  return (
    <section className="w-full">
      <PageHeader
        eyebrow="Account"
        title="Profile"
        description="Review your identity, role assignment, and recent account activity."
        icon={<UserRound className="h-5 w-5" aria-hidden="true" />}
        actions={
          <Link
            to="/profile/edit-nickname"
            className={buttonClassName({ variant: "secondary" })}
          >
            <Pencil className="h-4 w-4" aria-hidden="true" />
            Edit nickname
          </Link>
        }
      />

      <AccountTabs />
      {notice ? (
        <div className="mb-5">
          <Alert tone="success">{notice}</Alert>
        </div>
      ) : null}
      {loadError ? (
        <div className="mb-5">
          <Alert>{loadError}</Alert>
        </div>
      ) : null}

      {loading ? (
        <div className="surface-shadow flex min-h-44 items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white text-sm text-slate-600">
          <Spinner size="sm" /> Loading profile
        </div>
      ) : profile ? (
        <section
          className="surface-shadow overflow-hidden rounded-lg border border-slate-200 bg-white"
          aria-label="Account details"
        >
          <div className="flex flex-col gap-4 border-b border-slate-200 p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
            <div className="flex min-w-0 items-center gap-4">
              <span className="grid h-12 w-12 shrink-0 place-items-center rounded-md bg-slate-950 text-lg font-bold text-white">
                {initial}
              </span>
              <div className="min-w-0">
                <h2 className="truncate text-lg font-bold text-slate-950">
                  {displayName}
                </h2>
                <p className="truncate text-sm text-slate-500">@{profile.username}</p>
              </div>
            </div>
            <span
              className={
                profile.status === 1
                  ? "w-fit rounded-full bg-teal-50 px-2.5 py-1 text-xs font-bold text-teal-800"
                  : "w-fit rounded-full bg-red-50 px-2.5 py-1 text-xs font-bold text-red-700"
              }
            >
              {profile.status === 1 ? "Active" : "Disabled"}
            </span>
          </div>

          <dl className="grid sm:grid-cols-2">
            {[
              ["User ID", String(profile.id)],
              ["Username", profile.username],
              ["Nickname", profile.nickname || "Not set"],
              ["Last login", formatDateTime(profile.last_login_at)],
            ].map(([label, value], index) => (
              <div
                key={label}
                className={[
                  "p-5",
                  index < 2 ? "border-b border-slate-200" : "",
                  index % 2 === 0 ? "sm:border-r sm:border-slate-200" : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
              >
                <dt className="text-xs font-bold uppercase text-slate-500">{label}</dt>
                <dd className="mt-2 break-words text-sm font-semibold text-slate-900">
                  {value}
                </dd>
              </div>
            ))}
          </dl>

          <div className="border-t border-slate-200 bg-slate-50 px-5 py-4 sm:px-6">
            <p className="text-xs font-bold uppercase text-slate-500">Assigned roles</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {roleCodes.length > 0 ? (
                roleCodes.map((role) => (
                  <span
                    key={role}
                    className="rounded-md border border-slate-200 bg-white px-2.5 py-1 font-mono text-xs font-semibold text-slate-700"
                  >
                    {role}
                  </span>
                ))
              ) : (
                <span className="text-sm text-slate-500">No role assigned</span>
              )}
            </div>
          </div>
        </section>
      ) : null}
    </section>
  );
};

export default ProfilePage;
