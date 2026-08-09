import axios, { AxiosError, AxiosRequestConfig } from "axios";
import { normalizeApiError } from "./errors";
import { createRefreshCoordinator } from "./refresh-coordinator";
import { unwrap } from "./response";
import type { AccessSession, ApiResponse } from "./types";

const baseURL = import.meta.env.VITE_BACKEND_URL ?? "";

export const publicApi = axios.create({ baseURL, withCredentials: true });
export const privateApi = axios.create({ baseURL, withCredentials: true });

type AuthBridge = {
  getAccessToken: () => string | null;
  onTokenRefreshed: (session: AccessSession) => void;
  onSessionExpired: () => void;
};

let authBridge: AuthBridge = {
  getAccessToken: () => null,
  onTokenRefreshed: () => undefined,
  onSessionExpired: () => undefined,
};

export const configureApiAuth = (bridge: AuthBridge) => {
  authBridge = bridge;
};

const performRefresh = async (): Promise<AccessSession> => {
  const response = await publicApi.post<ApiResponse<AccessSession>>(
    "/api/v1/auth/refresh"
  );
  const session = unwrap(response.data);
  authBridge.onTokenRefreshed(session);
  return session;
};

export const refreshAccessToken = createRefreshCoordinator(performRefresh);

type RetryableRequest = AxiosRequestConfig & { _retry?: boolean };

privateApi.interceptors.request.use((config) => {
  const token = authBridge.getAccessToken();
  config.headers = config.headers ?? {};
  if (token && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

privateApi.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiResponse<null>>) => {
    const originalRequest = error.config as RetryableRequest | undefined;
    if (error.response?.status !== 401 || !originalRequest || originalRequest._retry) {
      throw normalizeApiError(error);
    }

    originalRequest._retry = true;
    try {
      const session = await refreshAccessToken();
      originalRequest.headers = originalRequest.headers ?? {};
      originalRequest.headers.Authorization = `Bearer ${session.access_token}`;
      return privateApi(originalRequest);
    } catch (refreshError) {
      authBridge.onSessionExpired();
      throw normalizeApiError(refreshError, "Your session has expired.");
    }
  }
);
