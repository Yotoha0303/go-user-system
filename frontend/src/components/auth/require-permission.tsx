import { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { selectPermissions } from "../../app/authSlice";
import { useAppSelector } from "../../app/hooks";

type RequirePermissionProps = {
  permission: string;
  children: ReactNode;
};

const RequirePermission = ({ permission, children }: RequirePermissionProps) => {
  const permissions = useAppSelector(selectPermissions);
  return permissions.includes(permission) ? (
    <>{children}</>
  ) : (
    <Navigate to="/forbidden" replace />
  );
};

export default RequirePermission;
