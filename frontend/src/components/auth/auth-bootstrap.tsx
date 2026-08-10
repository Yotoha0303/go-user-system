import { ReactNode, useEffect } from "react";
import { refreshAccessToken } from "../../api/client";
import { getCurrentUser, getMyAuthorization } from "../../api/user.api";
import {
  selectAuthStatus,
  sessionCleared,
  sessionRestored,
} from "../../app/authSlice";
import { useAppDispatch, useAppSelector } from "../../app/hooks";
import BrandMark from "../brand/brand-mark";
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
      <div className="grid min-h-screen place-items-center bg-[#f4f7f6] px-4">
        <div className="surface-shadow flex min-w-64 flex-col items-center rounded-lg border border-slate-200 bg-white px-8 py-7">
          <BrandMark />
          <div className="mt-5 flex items-center gap-2 text-sm font-medium text-slate-600">
            <Spinner size="sm" />
            Restoring session
          </div>
        </div>
      </div>
    );
  }

  return <>{children}</>;
};

export default AuthBootstrap;
