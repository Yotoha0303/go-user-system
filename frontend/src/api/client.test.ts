import axios, { AxiosResponse, InternalAxiosRequestConfig } from "axios";
import {
  configureApiAuth,
  privateApi,
  publicApi,
} from "./client";
import type { AccessSession, ApiResponse } from "./types";

const response = <T>(
  config: InternalAxiosRequestConfig,
  status: number,
  data: T
): AxiosResponse<T> => ({
  config,
  data,
  headers: {},
  status,
  statusText: String(status),
});

const rejectedResponse = (
  config: InternalAxiosRequestConfig,
  status: number,
  code: number
) =>
  Promise.reject(
    new axios.AxiosError(
      "request failed",
      "ERR_BAD_REQUEST",
      config,
      undefined,
      response<ApiResponse<null>>(config, status, {
        code,
        msg: "request failed",
        data: null,
      })
    )
  );

describe("privateApi authentication interceptors", () => {
  const originalPrivateAdapter = privateApi.defaults.adapter;
  const originalPublicAdapter = publicApi.defaults.adapter;

  afterEach(() => {
    privateApi.defaults.adapter = originalPrivateAdapter;
    publicApi.defaults.adapter = originalPublicAdapter;
    configureApiAuth({
      getAccessToken: () => null,
      onTokenRefreshed: () => undefined,
      onSessionExpired: () => undefined,
    });
  });

  it("uses one refresh for concurrent 401 responses and retries both requests", async () => {
    let accessToken = "expired-access-token";
    let refreshCalls = 0;
    let privateCalls = 0;
    const refreshedSession: AccessSession = {
      access_token: "new-access-token",
      access_token_expires_in: 900,
      refresh_token_expires_in: 604800,
    };
    configureApiAuth({
      getAccessToken: () => accessToken,
      onTokenRefreshed: (session) => {
        accessToken = session.access_token;
      },
      onSessionExpired: vi.fn(),
    });

    publicApi.defaults.adapter = async (config) => {
      refreshCalls += 1;
      await new Promise((resolve) => setTimeout(resolve, 5));
      return response<ApiResponse<AccessSession>>(config, 200, {
        code: 0,
        msg: "success",
        data: refreshedSession,
      });
    };
    privateApi.defaults.adapter = (config) => {
      privateCalls += 1;
      if (config.headers?.Authorization !== "Bearer new-access-token") {
        return rejectedResponse(config, 401, 3007);
      }
      return Promise.resolve(response(config, 200, { ok: true }));
    };

    const [first, second] = await Promise.all([
      privateApi.get("/protected/one"),
      privateApi.get("/protected/two"),
    ]);

    expect(first.data).toEqual({ ok: true });
    expect(second.data).toEqual({ ok: true });
    expect(refreshCalls).toBe(1);
    expect(privateCalls).toBe(4);
  });

  it("does not refresh a forbidden response", async () => {
    const refreshAdapter = vi.fn();
    publicApi.defaults.adapter = refreshAdapter;
    privateApi.defaults.adapter = (config) => rejectedResponse(config, 403, 5003);
    configureApiAuth({
      getAccessToken: () => "valid-access-token",
      onTokenRefreshed: vi.fn(),
      onSessionExpired: vi.fn(),
    });

    await expect(privateApi.get("/forbidden")).rejects.toMatchObject({
      status: 403,
      code: 5003,
    });
    expect(refreshAdapter).not.toHaveBeenCalled();
  });
});
