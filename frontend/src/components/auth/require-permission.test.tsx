import { configureStore } from "@reduxjs/toolkit";
import { render, screen } from "@testing-library/react";
import { Provider } from "react-redux";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import authReducer from "../../app/authSlice";
import RequirePermission from "./require-permission";

const renderGuard = (permissionCodes: string[]) => {
  const initialAuth = authReducer(undefined, { type: "test/init" });
  const store = configureStore({
    reducer: { auth: authReducer },
    preloadedState: {
      auth: {
        ...initialAuth,
        status: "authenticated" as const,
        permissionCodes,
      },
    },
  });
  render(
    <Provider store={store}>
      <MemoryRouter initialEntries={["/protected"]}>
        <Routes>
          <Route
            path="/protected"
            element={
              <RequirePermission permission="admin:roles:read">
                <div>Protected content</div>
              </RequirePermission>
            }
          />
          <Route path="/forbidden" element={<div>Forbidden page</div>} />
        </Routes>
      </MemoryRouter>
    </Provider>
  );
};

describe("RequirePermission", () => {
  it("renders protected content when permission exists", () => {
    renderGuard(["admin:roles:read"]);
    expect(screen.getByText("Protected content")).toBeInTheDocument();
  });

  it("routes to forbidden when permission is missing", () => {
    renderGuard([]);
    expect(screen.getByText("Forbidden page")).toBeInTheDocument();
  });
});
