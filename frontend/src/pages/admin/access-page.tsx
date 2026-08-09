import { Save, ShieldCheck } from "lucide-react";
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
import Spinner from "../../components/elements/spinner";

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

  return (
    <section className="w-full">
      <div className="mb-7 flex items-start gap-3">
        <div className="rounded-md bg-blue-50 p-2 text-blue-700">
          <ShieldCheck className="h-5 w-5" aria-hidden="true" />
        </div>
        <div>
          <p className="text-sm font-semibold text-blue-700">Administration</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-950">Access control</h1>
          <p className="mt-2 text-sm text-slate-600">Roles, permissions, and user role assignments.</p>
        </div>
      </div>

      {pageError ? <div className="mb-5"><Alert>{pageError}</Alert></div> : null}
      {loading ? (
        <div className="flex items-center gap-2 py-10 text-sm text-slate-600">
          <Spinner size="sm" /> Loading access data
        </div>
      ) : (
        <div className="space-y-10">
          <section aria-labelledby="roles-title">
            <h2 id="roles-title" className="mb-3 text-lg font-bold text-slate-900">Roles</h2>
            <div className="overflow-x-auto border-y border-slate-200 bg-white">
              <table className="w-full min-w-[32rem] text-left text-sm">
                <thead className="bg-slate-50 text-xs uppercase text-slate-500">
                  <tr><th className="px-4 py-3">ID</th><th className="px-4 py-3">Code</th><th className="px-4 py-3">Name</th></tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {roles.map((role) => (
                    <tr key={role.id}><td className="px-4 py-3 text-slate-500">{role.id}</td><td className="px-4 py-3 font-mono text-xs font-semibold text-slate-900">{role.code}</td><td className="px-4 py-3 text-slate-700">{role.name}</td></tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          {canReadPermissions ? (
            <section aria-labelledby="permissions-title">
              <h2 id="permissions-title" className="mb-3 text-lg font-bold text-slate-900">Permissions</h2>
              <div className="overflow-x-auto border-y border-slate-200 bg-white">
                <table className="w-full min-w-[44rem] text-left text-sm">
                  <thead className="bg-slate-50 text-xs uppercase text-slate-500">
                    <tr><th className="px-4 py-3">Code</th><th className="px-4 py-3">Name</th><th className="px-4 py-3">Method</th><th className="px-4 py-3">Path</th></tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200">
                    {permissions.map((permission) => (
                      <tr key={permission.id}><td className="px-4 py-3 font-mono text-xs font-semibold text-slate-900">{permission.code}</td><td className="px-4 py-3 text-slate-700">{permission.name}</td><td className="px-4 py-3 font-mono text-xs text-slate-600">{permission.method}</td><td className="px-4 py-3 font-mono text-xs text-slate-600">{permission.path}</td></tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          ) : null}

          {canAssignRoles ? (
            <section aria-labelledby="assign-title" className="max-w-2xl">
              <h2 id="assign-title" className="mb-3 text-lg font-bold text-slate-900">Assign user roles</h2>
              <form className="space-y-5 border-t border-slate-200 pt-5" onSubmit={handleAssign}>
                {saveError ? <Alert>{saveError}</Alert> : null}
                {notice ? <Alert tone="success">{notice}</Alert> : null}
                <div>
                  <label className="mb-1.5 block text-sm font-semibold text-slate-800" htmlFor="user-id">User ID</label>
                  <Input
                    id="user-id"
                    type="number"
                    min={1}
                    step={1}
                    inputMode="numeric"
                    value={userID}
                    onChange={(event) => setUserID(event.target.value)}
                  />
                </div>
                <fieldset>
                  <legend className="mb-2 text-sm font-semibold text-slate-800">Roles</legend>
                  <div className="flex flex-wrap gap-3">
                    {roles.map((role) => (
                      <label key={role.id} className="flex min-h-10 cursor-pointer items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm font-medium text-slate-800">
                        <input type="checkbox" className="h-4 w-4 accent-blue-600" checked={selectedRoles.includes(role.code)} onChange={() => toggleRole(role.code)} />
                        {role.name}
                      </label>
                    ))}
                  </div>
                </fieldset>
                <Button type="submit" isLoading={saving} icon={<Save className="h-4 w-4" aria-hidden="true" />}>Update roles</Button>
              </form>
            </section>
          ) : null}
        </div>
      )}
    </section>
  );
};

export default AccessPage;
