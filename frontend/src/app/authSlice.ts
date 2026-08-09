import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import type { RootState } from "./store";
import type { UserProfile } from "../api/types";

export type AuthStatus = "initializing" | "authenticated" | "anonymous";

export type AuthState = {
  status: AuthStatus;
  accessToken: string | null;
  accessTokenExpiresAt: number | null;
  user: UserProfile | null;
  roleCodes: string[];
  permissionCodes: string[];
};

const initialState: AuthState = {
  status: "initializing",
  accessToken: null,
  accessTokenExpiresAt: null,
  user: null,
  roleCodes: [],
  permissionCodes: [],
};

type SessionPayload = {
  accessToken: string;
  accessTokenExpiresAt: number;
  user: UserProfile;
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    sessionStarted(state, action: PayloadAction<SessionPayload>) {
      state.status = "authenticated";
      state.accessToken = action.payload.accessToken;
      state.accessTokenExpiresAt = action.payload.accessTokenExpiresAt;
      state.user = action.payload.user;
    },
    sessionRestored(
      state,
      action: PayloadAction<{
        user: UserProfile;
        roleCodes: string[];
        permissionCodes: string[];
      }>
    ) {
      state.status = "authenticated";
      state.user = action.payload.user;
      state.roleCodes = action.payload.roleCodes;
      state.permissionCodes = action.payload.permissionCodes;
    },
    tokenRefreshed(
      state,
      action: PayloadAction<{ accessToken: string; accessTokenExpiresAt: number }>
    ) {
      state.accessToken = action.payload.accessToken;
      state.accessTokenExpiresAt = action.payload.accessTokenExpiresAt;
    },
    authorizationLoaded(
      state,
      action: PayloadAction<{ roleCodes: string[]; permissionCodes: string[] }>
    ) {
      state.roleCodes = action.payload.roleCodes;
      state.permissionCodes = action.payload.permissionCodes;
    },
    profileUpdated(state, action: PayloadAction<UserProfile>) {
      state.user = action.payload;
    },
    sessionCleared(state) {
      state.status = "anonymous";
      state.accessToken = null;
      state.accessTokenExpiresAt = null;
      state.user = null;
      state.roleCodes = [];
      state.permissionCodes = [];
    },
  },
});

export const {
  sessionStarted,
  sessionRestored,
  tokenRefreshed,
  authorizationLoaded,
  profileUpdated,
  sessionCleared,
} = authSlice.actions;

export default authSlice.reducer;

export const selectAuth = (state: RootState) => state.auth;
export const selectAuthStatus = (state: RootState) => state.auth.status;
export const selectCurrentUser = (state: RootState) => state.auth.user;
export const selectPermissions = (state: RootState) => state.auth.permissionCodes;
export const selectHasPermission = (permission: string) => (state: RootState) =>
  state.auth.permissionCodes.includes(permission);
