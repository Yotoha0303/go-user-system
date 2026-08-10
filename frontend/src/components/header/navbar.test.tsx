import { configureStore } from "@reduxjs/toolkit";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Provider } from "react-redux";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { logout } from "../../api/auth.api";
import authReducer, { sessionStarted } from "../../app/authSlice";
import Navbar from "./navbar";

vi.mock("../../api/auth.api", () => ({ logout: vi.fn() }));

describe("Navbar", () => {
  it("opens and closes the mobile navigation drawer", async () => {
    const store = configureStore({ reducer: { auth: authReducer } });
    store.dispatch(
      sessionStarted({
        accessToken: "access-token",
        accessTokenExpiresAt: Date.now() + 10000,
        user: { id: 1, username: "alice", nickname: "Alice", status: 1 },
      })
    );
    const user = userEvent.setup();
    render(
      <Provider store={store}>
        <MemoryRouter initialEntries={["/profile"]}>
          <Navbar />
        </MemoryRouter>
      </Provider>
    );

    const openButton = screen.getByRole("button", { name: "Open navigation" });
    const sidebar = document.getElementById("primary-sidebar");
    expect(openButton).toHaveAttribute("aria-expanded", "false");
    expect(sidebar).toHaveClass("invisible", "overflow-y-auto");

    await user.click(openButton);
    expect(openButton).toHaveAttribute("aria-expanded", "true");
    expect(sidebar).toHaveClass("visible");
    expect(sidebar).not.toHaveClass("invisible");

    await user.click(screen.getAllByRole("button", { name: "Close navigation" })[0]);
    expect(openButton).toHaveAttribute("aria-expanded", "false");
    expect(sidebar).toHaveClass("invisible");
  });

  it("clears local session even when server logout fails", async () => {
    vi.mocked(logout).mockRejectedValue(new Error("network unavailable"));
    const store = configureStore({ reducer: { auth: authReducer } });
    store.dispatch(
      sessionStarted({
        accessToken: "access-token",
        accessTokenExpiresAt: Date.now() + 10000,
        user: { id: 1, username: "alice", nickname: "Alice", status: 1 },
      })
    );
    const user = userEvent.setup();
    render(
      <Provider store={store}>
        <MemoryRouter initialEntries={["/profile"]}>
          <Navbar />
          <Routes>
            <Route path="/profile" element={<div>Profile page</div>} />
            <Route path="/auth/login" element={<div>Login destination</div>} />
          </Routes>
        </MemoryRouter>
      </Provider>
    );

    await act(async () => {
      await user.click(screen.getByRole("button", { name: "Sign out" }));
    });
    await screen.findByText("Login destination");
    await waitFor(() => expect(logout).toHaveBeenCalledWith("access-token"));
    await waitFor(() => expect(store.getState().auth.status).toBe("anonymous"));
  });
});
