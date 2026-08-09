import { configureStore } from "@reduxjs/toolkit";
import { configureApiAuth } from "../api/client";
import authReducer, { sessionCleared, tokenRefreshed } from "./authSlice";

export const store = configureStore({
  reducer: { auth: authReducer },
  devTools: import.meta.env.DEV,
});

configureApiAuth({
  getAccessToken: () => store.getState().auth.accessToken,
  onTokenRefreshed: (session) => {
    store.dispatch(
      tokenRefreshed({
        accessToken: session.access_token,
        accessTokenExpiresAt: Date.now() + session.access_token_expires_in * 1000,
      })
    );
  },
  onSessionExpired: () => store.dispatch(sessionCleared()),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
