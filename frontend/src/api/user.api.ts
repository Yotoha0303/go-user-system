import { privateApi } from "./client";
import { unwrap } from "./response";
import type { ApiResponse, AuthorizationInfo, UserProfile } from "./types";

export const getCurrentUser = async (): Promise<UserProfile> => {
  const response = await privateApi.get<ApiResponse<UserProfile>>("/api/v1/users/me");
  return unwrap(response.data);
};

export const getMyAuthorization = async (): Promise<AuthorizationInfo> => {
  const response = await privateApi.get<ApiResponse<AuthorizationInfo>>(
    "/api/v1/users/me/authorization"
  );
  return unwrap(response.data);
};

export const updateNickname = async (nickname: string): Promise<void> => {
  const response = await privateApi.put<ApiResponse<null>>(
    "/api/v1/users/me/profile",
    { nickname }
  );
  unwrap(response.data);
};

export const updatePassword = async (
  oldPassword: string,
  newPassword: string
): Promise<void> => {
  const response = await privateApi.patch<ApiResponse<null>>(
    "/api/v1/users/me/update/password",
    { old_password: oldPassword, new_password: newPassword }
  );
  unwrap(response.data);
};
