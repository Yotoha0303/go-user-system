import {
  KeyRound,
  Save,
  ShieldCheck,
  UserCog,
  UsersRound,
} from "lucide-react";
import { FormEvent, useEffect, useState } from "react";
import { errorMessage } from "../../api/errors";
import { assignUserRoles, listPermissions, listRoles } from "../../api/rbac.api";
import type { Permission, Role } from "../../api/types";
import { PermissionCode } from "../../api/types";
import { selectPermissions } from "../../app/authSlice";
import { useAppSelector } from "../../app/hooks";
import Alert from "../../components/elements/alert";
import Button from "../../components/elements/button";
import Input from "../../components/elements/input";
import PageHeader from "../../components/elements/page-header";
import Spinner from "../../components/elements/spinner";

const methodClassName = (method: string) => {
  switch (method.toUpperCase()) {
    case "GET":
      return "bg-teal-50 text-teal-800";
    case "DELETE":
      return "bg-red-50 text-red-700";
    case "POST":
    case "PUT":
    case "PATCH":
      return "bg-amber-50 text-amber-800";
    default:
      return "bg-slate-100 text-slate-700";
  }
};

const AccessPage = () => {
  const ownPermissions = useAppSelector(selectPermissions);
  const canReadPermissions = ownPermissions.includes(
    PermissionCode.adminPermissionsRead
  );
  const canAssignRoles = ownPermissions.includes(
    PermissionCode.adminUserRolesUpdate
  );
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(true);
  const [pageError, setPageError] = useState("");
  const [userID, setUserID] = useState("");
  const [selectedRoles, setSelectedRoles] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [notice, setNotice] = useState("");

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const [roleData, permissionData] = await Promise.all([
          listRoles(),
          canReadPermissions ? listPermissions() : Promise.resolve([]),
        ]);
        if (!active) return;
        setRoles(roleData);
        setPermissions(permissionData);
      } catch (error) {
        if (active) setPageError(errorMessage(error, "Unable to load access data."));
      } finally {
        if (active) setLoading(false);
      }
    };
    void load();
    return () => {
      active = false;
    };
  }, [canReadPermissions]);

  const toggleRole = (roleCode: string) => {
    setSelectedRoles((current) =>
      current.includes(roleCode)
        ? current.filter((code) => code !== roleCode)
        : [...current, roleCode]
    );
  };

  const handleAssign = async (event: FormEvent) => {
    event.preventDefault();
    setSaveError("");
    setNotice("");
    const parsedUserID = Number(userID);
    if (!Number.isInteger(parsedUserID) || parsedUserID <= 0) {
      setSaveError("Enter a valid user ID.");
      return;
    }
    if (selectedRoles.length === 0) {
      setSaveError("Select at least one role.");
      return;
    }

    setSaving(true);
    try {
      await assignUserRoles(parsedUserID, selectedRoles);
      setNotice(`Roles updated for user ${parsedUserID}.`);
    } catch (error) {
      setSaveError(errorMessage(error, "Unable to assign roles."));
    } finally {
      setSaving(false);
    }
  };

  const metrics = [
    {
      label: "Roles",
      value: loading ? "--" : String(roles.length),
      icon: <UsersRound className="h-4 w-4" aria-hidden="true" />,
    },
    {
      label: "Permissions",
      value: loading || !canReadPermissions ? "--" : String(permissions.length),
      icon: <KeyRound className="h-4 w-4" aria-hidden="true" />,
    },
    {
      label: "Assignment",
      value: canAssignRoles ? "Enabled" : "Read only",
      icon: <UserCog className="h-4 w-4" aria-hidden="true" />,
    },
  ];

  return (
    <section className="w-full">
      <PageHeader
        eyebrow="Administration"
        title="Access control"
        description="Inspect authorization definitions and manage role assignments."
        icon={<ShieldCheck className="h-5 w-5" aria-hidden="true" />}
      />

      <div className="mb-6 grid gap-3 sm:grid-cols-3">
        {metrics.map((metric) => (
          <div
            key={metric.label}
            className="surface-shadow flex min-h-24 items-center gap-4 rounded-lg border border-slate-200 bg-white p-4"
          >
            <span className="grid h-9 w-9 shrink-0 place-items-center rounded-md bg-slate-100 text-slate-700">
              {metric.icon}
            </span>
            <div className="min-w-0">
              <p className="text-xs font-bold uppercase text-slate-500">{metric.label}</p>
              <p className="mt-1 truncate text-xl font-bold text-slate-950">
                {metric.value}
              </p>
            </div>
          </div>
        ))}
      </div>

      {pageError ? (
        <div className="mb-5">
          <Alert>{pageError}</Alert>
        </div>
      ) : null}
      {loading ? (
        <div className="surface-shadow flex min-h-44 items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white text-sm text-slate-600">
          <Spinner size="sm" /> Loading access data
        </div>
      ) : (
        <div className="space-y-6">
          <div
            className={
              canAssignRoles
                ? "grid items-start gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(20rem,0.8fr)]"
                : ""
            }
          >
            <section
              aria-labelledby="roles-title"
              className="surface-shadow overflow-hidden rounded-lg border border-slate-200 bg-white"
            >
              <div className="border-b border-slate-200 px-5 py-4">
                <h2 id="roles-title" className="text-sm font-bold text-slate-950">
                  Roles
                </h2>
                <p className="mt-1 text-sm text-slate-500">
                  Role codes currently available to the authorization service.
                </p>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[32rem] text-left text-sm">
                  <thead className="bg-slate-50 text-xs uppercase text-slate-500">
                    <tr>
                      <th className="px-5 py-3 font-bold">ID</th>
                      <th className="px-5 py-3 font-bold">Code</th>
                      <th className="px-5 py-3 font-bold">Name</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200">
                    {roles.length > 0 ? (
                      roles.map((role) => (
                        <tr key={role.id} className="hover:bg-slate-50">
                          <td className="px-5 py-3 text-slate-500">{role.id}</td>
                          <td className="px-5 py-3 font-mono text-xs font-semibold text-slate-900">
                            {role.code}
                          </td>
                          <td className="px-5 py-3 text-slate-700">{role.name}</td>
                        </tr>
                      ))
                    ) : (
                      <tr>
                        <td colSpan={3} className="px-5 py-10 text-center text-slate-500">
                          No roles found.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </section>

            {canAssignRoles ? (
              <section
                aria-labelledby="assign-title"
                className="surface-shadow overflow-hidden rounded-lg border border-slate-200 bg-white"
              >
                <div className="border-b border-slate-200 px-5 py-4">
                  <h2 id="assign-title" className="text-sm font-bold text-slate-950">
                    Assign user roles
                  </h2>
                  <p className="mt-1 text-sm text-slate-500">
                    Replace a user's role assignment.
                  </p>
                </div>
                <form className="space-y-5 p-5" onSubmit={handleAssign}>
                  {saveError ? <Alert>{saveError}</Alert> : null}
                  {notice ? <Alert tone="success">{notice}</Alert> : null}
                  <div>
                    <label
                      className="mb-2 block text-sm font-semibold text-slate-800"
                      htmlFor="user-id"
                    >
                      User ID
                    </label>
                    <Input
                      id="user-id"
                      type="number"
                      min={1}
                      step={1}
                      inputMode="numeric"
                      placeholder="Enter a numeric ID"
                      value={userID}
                      onChange={(event) => setUserID(event.target.value)}
                    />
                  </div>
                  <fieldset>
                    <legend className="mb-2 text-sm font-semibold text-slate-800">
                      Roles
                    </legend>
                    <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
                      {roles.map((role) => {
                        const selected = selectedRoles.includes(role.code);
                        return (
                          <label
                            key={role.id}
                            className={
                              selected
                                ? "flex min-h-11 cursor-pointer items-center gap-3 rounded-md border border-teal-300 bg-teal-50 px-3 text-sm font-semibold text-teal-950"
                                : "flex min-h-11 cursor-pointer items-center gap-3 rounded-md border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 hover:border-slate-300"
                            }
                          >
                            <input
                              type="checkbox"
                              className="h-4 w-4 shrink-0 accent-teal-700"
                              checked={selected}
                              onChange={() => toggleRole(role.code)}
                            />
                            <span className="min-w-0">
                              <span className="block truncate">{role.name}</span>
                              <span className="block truncate font-mono text-xs font-normal opacity-70">
                                {role.code}
                              </span>
                            </span>
                          </label>
                        );
                      })}
                    </div>
                  </fieldset>
                  <Button
                    type="submit"
                    isLoading={saving}
                    icon={<Save className="h-4 w-4" aria-hidden="true" />}
                  >
                    Update roles
                  </Button>
                </form>
              </section>
            ) : null}
          </div>

          {canReadPermissions ? (
            <section
              aria-labelledby="permissions-title"
              className="surface-shadow overflow-hidden rounded-lg border border-slate-200 bg-white"
            >
              <div className="border-b border-slate-200 px-5 py-4">
                <h2 id="permissions-title" className="text-sm font-bold text-slate-950">
                  Permissions
                </h2>
                <p className="mt-1 text-sm text-slate-500">
                  API operations protected by the current RBAC policy.
                </p>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[48rem] text-left text-sm">
                  <thead className="bg-slate-50 text-xs uppercase text-slate-500">
                    <tr>
                      <th className="px-5 py-3 font-bold">Code</th>
                      <th className="px-5 py-3 font-bold">Name</th>
                      <th className="px-5 py-3 font-bold">Method</th>
                      <th className="px-5 py-3 font-bold">Path</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200">
                    {permissions.length > 0 ? (
                      permissions.map((permission) => (
                        <tr key={permission.id} className="hover:bg-slate-50">
                          <td className="px-5 py-3 font-mono text-xs font-semibold text-slate-900">
                            {permission.code}
                          </td>
                          <td className="px-5 py-3 text-slate-700">{permission.name}</td>
                          <td className="px-5 py-3">
                            <span
                              className={`inline-flex rounded-md px-2 py-1 font-mono text-xs font-bold ${methodClassName(
                                permission.method
                              )}`}
                            >
                              {permission.method}
                            </span>
                          </td>
                          <td className="px-5 py-3 font-mono text-xs text-slate-600">
                            {permission.path}
                          </td>
                        </tr>
                      ))
                    ) : (
                      <tr>
                        <td
                          colSpan={4}
                          className="px-5 py-10 text-center text-slate-500"
                        >
                          No permissions found.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </section>
          ) : null}
        </div>
      )}
    </section>
  );
};

export default AccessPage;
