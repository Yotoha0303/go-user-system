import { yupResolver } from "@hookform/resolvers/yup";
import { UserPlus } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";
import * as yup from "yup";
import { register as registerAccount } from "../../api/auth.api";
import { errorMessage } from "../../api/errors";
import Alert from "../../components/elements/alert";
import Button from "../../components/elements/button";
import FieldError from "../../components/elements/field-error";
import Input from "../../components/elements/input";
import AuthLayout from "../../components/layouts/auth-layout";

type SignupFields = {
  username: string;
  password: string;
  confirmPassword: string;
};

const schema = yup
  .object({
    username: yup
      .string()
      .trim()
      .min(3, "Username must be at least 3 characters")
      .max(64, "Username must be at most 64 characters")
      .required("Username is required"),
    password: yup
      .string()
      .min(12, "Password must be at least 12 characters")
      .max(72, "Password must be at most 72 characters")
      .required("Password is required"),
    confirmPassword: yup
      .string()
      .oneOf([yup.ref("password")], "Passwords must match")
      .required("Please confirm your password"),
  })
  .required();

const SignupPage = () => {
  const [submissionError, setSubmissionError] = useState("");
  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<SignupFields>({ resolver: yupResolver(schema) });

  const onSubmit = async (fields: SignupFields) => {
    setSubmissionError("");
    try {
      await registerAccount({
        username: fields.username.trim(),
        password: fields.password,
      });
      navigate("/auth/login", {
        replace: true,
        state: { notice: "Account created. You can now sign in." },
      });
    } catch (error) {
      setSubmissionError(errorMessage(error, "Unable to create account."));
    }
  };

  return (
    <AuthLayout
      title="Create account"
      subtitle="Set up a username and password for your account."
      footer={
        <>
          Already registered?{" "}
          <Link className="font-semibold text-teal-700 hover:text-teal-900" to="/auth/login">
            Sign in
          </Link>
        </>
      }
    >
      <form className="space-y-5" noValidate onSubmit={handleSubmit(onSubmit)}>
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
            autoComplete="new-password"
            aria-invalid={Boolean(errors.password)}
            aria-describedby={errors.password ? "password-error" : undefined}
            {...register("password")}
          />
          <FieldError id="password-error" message={errors.password?.message} />
        </div>

        <div>
          <label className="mb-2 block text-sm font-semibold text-slate-800" htmlFor="confirm-password">
            Confirm password
          </label>
          <Input
            id="confirm-password"
            type="password"
            autoComplete="new-password"
            aria-invalid={Boolean(errors.confirmPassword)}
            aria-describedby={errors.confirmPassword ? "confirm-password-error" : undefined}
            {...register("confirmPassword")}
          />
          <FieldError id="confirm-password-error" message={errors.confirmPassword?.message} />
        </div>

        <Button
          className="w-full"
          type="submit"
          isLoading={isSubmitting}
          icon={<UserPlus className="h-4 w-4" aria-hidden="true" />}
        >
          Create account
        </Button>
      </form>
    </AuthLayout>
  );
};

export default SignupPage;
