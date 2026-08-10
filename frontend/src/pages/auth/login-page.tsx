import { yupResolver } from "@hookform/resolvers/yup";
import { LogIn } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useLocation, useNavigate } from "react-router-dom";
import * as yup from "yup";
import { login, logout } from "../../api/auth.api";
import { errorMessage } from "../../api/errors";
import { getMyAuthorization } from "../../api/user.api";
import {
  authorizationLoaded,
  sessionCleared,
  sessionStarted,
} from "../../app/authSlice";
import { useAppDispatch } from "../../app/hooks";
import Alert from "../../components/elements/alert";
import Button from "../../components/elements/button";
import FieldError from "../../components/elements/field-error";
import Input from "../../components/elements/input";
import AuthLayout from "../../components/layouts/auth-layout";

type LoginFields = { username: string; password: string };
type LocationState = { from?: string; notice?: string };

const schema = yup
  .object({
    username: yup.string().trim().required("Username is required"),
    password: yup.string().required("Password is required"),
  })
  .required();

const LoginPage = () => {
  const [submissionError, setSubmissionError] = useState("");
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const location = useLocation();
  const locationState = location.state as LocationState | null;
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFields>({ resolver: yupResolver(schema) });

  const onSubmit = async (fields: LoginFields) => {
    setSubmissionError("");
    try {
      const session = await login({
        username: fields.username.trim(),
        password: fields.password,
      });
      dispatch(
        sessionStarted({
          accessToken: session.access_token,
          accessTokenExpiresAt:
            Date.now() + session.access_token_expires_in * 1000,
          user: session.user,
        })
      );

      try {
        const authorization = await getMyAuthorization();
        dispatch(
          authorizationLoaded({
            roleCodes: authorization.role_codes,
            permissionCodes: authorization.permission_codes,
          })
        );
      } catch (authorizationError) {
        dispatch(sessionCleared());
        await logout().catch(() => undefined);
        throw authorizationError;
      }

      navigate(locationState?.from || "/profile", { replace: true });
    } catch (error) {
      setSubmissionError(errorMessage(error, "Unable to sign in."));
    }
  };

  return (
    <AuthLayout
      title="Sign in"
      subtitle="Use your account credentials to continue."
      footer={
        <>
          New here?{" "}
          <Link className="font-semibold text-teal-700 hover:text-teal-900" to="/auth/signup">
            Create an account
          </Link>
        </>
      }
    >
      <form className="space-y-5" noValidate onSubmit={handleSubmit(onSubmit)}>
        {locationState?.notice ? <Alert tone="success">{locationState.notice}</Alert> : null}
        {submissionError ? <Alert>{submissionError}</Alert> : null}

        <div>
          <label className="mb-2 block text-sm font-semibold text-slate-800" htmlFor="username">
            Username
          </label>
          <Input
            id="username"
            autoComplete="username"
            aria-invalid={Boolean(errors.username)}
            aria-describedby={errors.username ? "username-error" : undefined}
            {...register("username")}
          />
          <FieldError id="username-error" message={errors.username?.message} />
        </div>

        <div>
          <label className="mb-2 block text-sm font-semibold text-slate-800" htmlFor="password">
            Password
          </label>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            aria-invalid={Boolean(errors.password)}
            aria-describedby={errors.password ? "password-error" : undefined}
            {...register("password")}
          />
          <FieldError id="password-error" message={errors.password?.message} />
        </div>

        <Button
          className="w-full"
          type="submit"
          isLoading={isSubmitting}
          icon={<LogIn className="h-4 w-4" aria-hidden="true" />}
        >
          Sign in
        </Button>
      </form>
    </AuthLayout>
  );
};

export default LoginPage;
