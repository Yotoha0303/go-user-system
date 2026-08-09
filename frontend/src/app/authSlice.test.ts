import authReducer, {
  authorizationLoaded,
  sessionCleared,
  sessionStarted,
  tokenRefreshed,
} from "./authSlice";

describe("authSlice", () => {
  it("establishes, refreshes, and clears a session", () => {
    const user = {
      id: 1,
      username: "alice",
      nickname: "Alice",
      status: 1,
    };
    let state = authReducer(
      undefined,
      sessionStarted({
        accessToken: "access-1",
        accessTokenExpiresAt: 1000,
        user,
      })
    );
    expect(state.status).toBe("authenticated");
    expect(state.user).toEqual(user);

    state = authReducer(
      state,
      authorizationLoaded({
        roleCodes: ["user"],
        permissionCodes: ["profile:read"],
      })
    );
    state = authReducer(
      state,
      tokenRefreshed({ accessToken: "access-2", accessTokenExpiresAt: 2000 })
    );
    expect(state.accessToken).toBe("access-2");
    expect(state.permissionCodes).toContain("profile:read");

    state = authReducer(state, sessionCleared());
    expect(state.status).toBe("anonymous");
    expect(state.accessToken).toBeNull();
    expect(state.permissionCodes).toEqual([]);
  });
});
