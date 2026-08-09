import { configureStore } from "@reduxjs/toolkit";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Provider } from "react-redux";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { login, logout } from "../../api/auth.api";
import { getMyAuthorization } from "../../api/user.api";
import authReducer from "../../app/authSlice";
import LoginPage from "./login-page";

vi.mock("../../api/auth.api", () => ({ login: vi.fn(), logout: vi.fn() }));
vi.mock("../../api/user.api", () => ({ getMyAuthorization: vi.fn() }));

describe("LoginPage", () => {
  it("creates the authenticated user and authorization state", async () => {
    vi.mocked(login).mockResolvedValue({
      access_token: "access-token",
      access_token_expires_in: 900,
      refresh_token_expires_in: 604800,
      user: { id: 1, username: "alice", nickname: "Alice", status: 1 },
    });
    vi.mocked(getMyAuthorization).mockResolvedValue({
      role_codes: ["user"],
      permission_codes: ["profile:read"],
    });
    vi.mocked(logout).mockResolvedValue();

    const store = configureStore({ reducer: { auth: authReducer } });
    const user = userEvent.setup();
    render(
      <Provider store={store}>
        <MemoryRouter initialEntries={["/auth/login"]}>
          <Routes>
            <Route path="/auth/login" element={<LoginPage />} />
            <Route path="/profile" element={<div>Profile destination</div>} />
          </Routes>
        </MemoryRouter>
      </Provider>
    );

    await act(async () => {
      await user.type(screen.getByLabelText("Username"), "alice");
      await user.type(screen.getByLabelText("Password"), "secret1");
      await user.click(screen.getByRole("button", { name: "Sign in" }));
      await waitFor(() => expect(store.getState().auth.status).toBe("authenticated"));
    });

    await screen.findByText("Profile destination");
    expect(store.getState().auth.permissionCodes).toEqual(["profile:read"]);
    expect(login).toHaveBeenCalledWith({ username: "alice", password: "secret1" });
  });
});
