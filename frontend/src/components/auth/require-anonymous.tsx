import { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { selectAuthStatus } from "../../app/authSlice";
import { useAppSelector } from "../../app/hooks";

const RequireAnonymous = ({ children }: { children: ReactNode }) => {
  const status = useAppSelector(selectAuthStatus);
  return status === "authenticated" ? <Navigate to="/profile" replace /> : <>{children}</>;
};

export default RequireAnonymous;
