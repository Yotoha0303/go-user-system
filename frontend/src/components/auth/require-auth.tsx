import { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { selectAuthStatus } from "../../app/authSlice";
import { useAppSelector } from "../../app/hooks";

const RequireAuth = ({ children }: { children: ReactNode }) => {
  const status = useAppSelector(selectAuthStatus);
  const location = useLocation();

  if (status !== "authenticated") {
    return <Navigate to="/auth/login" replace state={{ from: location.pathname }} />;
  }
  return <>{children}</>;
};

export default RequireAuth;
