import { publicApi } from "./client";
import { unwrap } from "./response";
import type { ApiResponse, LoginSession } from "./types";

export type Credentials = { username: string; password: string };

export const login = async (credentials: Credentials): Promise<LoginSession> => {
  const response = await publicApi.post<ApiResponse<LoginSession>>(
    "/api/v1/auth/login",
    credentials
  );
  return unwrap(response.data);
};

export const register = async (credentials: Credentials): Promise<void> => {
  const response = await publicApi.post<ApiResponse<null>>(
    "/api/v1/auth/register",
    credentials
  );
  unwrap(response.data);
};

export const logout = async (): Promise<void> => {
  const response = await publicApi.post<ApiResponse<null>>("/api/v1/auth/logout");
  unwrap(response.data);
};
