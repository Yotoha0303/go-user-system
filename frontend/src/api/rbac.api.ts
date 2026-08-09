import { privateApi } from "./client";
import { unwrap } from "./response";
import type { ApiResponse, Permission, Role } from "./types";

export const listRoles = async (): Promise<Role[]> => {
  const response = await privateApi.get<ApiResponse<Role[]>>("/api/v1/admin/roles");
  return unwrap(response.data);
};

export const listPermissions = async (): Promise<Permission[]> => {
  const response = await privateApi.get<ApiResponse<Permission[]>>(
    "/api/v1/admin/permissions"
  );
  return unwrap(response.data);
};

export const assignUserRoles = async (
  userID: number,
  roleCodes: string[]
): Promise<void> => {
  const response = await privateApi.put<ApiResponse<null>>(
    `/api/v1/admin/users/${userID}/roles`,
    { role_codes: roleCodes }
  );
  unwrap(response.data);
};
