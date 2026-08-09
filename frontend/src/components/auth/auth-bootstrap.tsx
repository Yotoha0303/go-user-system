import { ReactNode, useEffect } from "react";
import { refreshAccessToken } from "../../api/client";
import { getCurrentUser, getMyAuthorization } from "../../api/user.api";
import {
  selectAuthStatus,
  sessionCleared,
  sessionRestored,
} from "../../app/authSlice";
import { useAppDispatch, useAppSelector } from "../../app/hooks";
import Spinner from "../elements/spinner";

const AuthBootstrap = ({ children }: { children: ReactNode }) => {
  const status = useAppSelector(selectAuthStatus);
  const dispatch = useAppDispatch();

  useEffect(() => {
    if (status !== "initializing") return;

    let active = true;
    const restore = async () => {
      try {
        await refreshAccessToken();
        const [user, authorization] = await Promise.all([
          getCurrentUser(),
          getMyAuthorization(),
        ]);
        if (!active) return;
        dispatch(
          sessionRestored({
            user,
            roleCodes: authorization.role_codes,
            permissionCodes: authorization.permission_codes,
          })
        );
      } catch {
        if (active) dispatch(sessionCleared());
      }
    };

    void restore();
    return () => {
      active = false;
    };
  }, [dispatch, status]);

  if (status === "initializing") {
    return (
      <div className="grid min-h-screen place-items-center bg-slate-50">
        <div className="flex items-center gap-3 text-sm font-medium text-slate-600">
          <Spinner size="sm" />
          Restoring session
        </div>
      </div>
    );
  }

  return <>{children}</>;
};

export default AuthBootstrap;
