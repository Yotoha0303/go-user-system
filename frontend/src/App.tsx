import { Route, Routes } from "react-router-dom";
import { PermissionCode } from "./api/types";
import RequireAnonymous from "./components/auth/require-anonymous";
import RequireAuth from "./components/auth/require-auth";
import RequirePermission from "./components/auth/require-permission";
import MainLayout from "./components/layouts/main-layout";
import AccessPage from "./pages/admin/access-page";
import LoginPage from "./pages/auth/login-page";
import SignupPage from "./pages/auth/signup-page";
import ForbiddenPage from "./pages/forbidden-page";
import HomePage from "./pages/home";
import NotFoundPage from "./pages/not-found-page";
import EditNicknamePage from "./pages/profile/edit-nickname-page";
import ProfilePage from "./pages/profile/profile-page";
import PasswordPage from "./pages/security/password-page";

const App = () => (
  <Routes>
    <Route element={<MainLayout />}>
      <Route index element={<HomePage />} />
      <Route
        path="auth/login"
        element={<RequireAnonymous><LoginPage /></RequireAnonymous>}
      />
      <Route
        path="auth/signup"
        element={<RequireAnonymous><SignupPage /></RequireAnonymous>}
      />
      <Route path="profile" element={<RequireAuth><ProfilePage /></RequireAuth>} />
      <Route
        path="profile/edit-nickname"
        element={<RequireAuth><RequirePermission permission={PermissionCode.profileUpdate}><EditNicknamePage /></RequirePermission></RequireAuth>}
      />
      <Route
        path="security/password"
        element={<RequireAuth><RequirePermission permission={PermissionCode.passwordUpdate}><PasswordPage /></RequirePermission></RequireAuth>}
      />
      <Route
        path="admin/access"
        element={<RequireAuth><RequirePermission permission={PermissionCode.adminRolesRead}><AccessPage /></RequirePermission></RequireAuth>}
      />
      <Route path="forbidden" element={<ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Route>
  </Routes>
);

export default App;
