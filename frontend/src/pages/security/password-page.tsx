import { yupResolver } from "@hookform/resolvers/yup";
import { KeyRound } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import * as yup from "yup";
import { errorMessage } from "../../api/errors";
import { updatePassword } from "../../api/user.api";
import { sessionCleared } from "../../app/authSlice";
import { useAppDispatch } from "../../app/hooks";
import AccountTabs from "../../components/account/account-tabs";
import Alert from "../../components/elements/alert";
import Button from "../../components/elements/button";
import FieldError from "../../components/elements/field-error";
import Input from "../../components/elements/input";

type PasswordFields = {
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
};

const schema = yup
  .object({
    oldPassword: yup.string().required("Current password is required"),
    newPassword: yup
      .string()
      .min(12, "Password must be at least 12 characters")
      .max(72, "Password must be at most 72 characters")
      .notOneOf([yup.ref("oldPassword")], "New password must be different")
      .required("New password is required"),
    confirmPassword: yup
      .string()
      .oneOf([yup.ref("newPassword")], "Passwords must match")
      .required("Please confirm your new password"),
  })
  .required();

const PasswordPage = () => {
  const [submissionError, setSubmissionError] = useState("");
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<PasswordFields>({ resolver: yupResolver(schema) });

  const onSubmit = async (fields: PasswordFields) => {
    setSubmissionError("");
    try {
      await updatePassword(fields.oldPassword, fields.newPassword);
      dispatch(sessionCleared());
      navigate("/auth/login", {
        replace: true,
        state: { notice: "Password updated. Sign in again with your new password." },
      });
    } catch (error) {
      setSubmissionError(errorMessage(error, "Unable to update password."));
    }
  };

  return (
    <section className="mx-auto w-full max-w-3xl">
      <div className="mb-6">
        <p className="text-sm font-semibold text-blue-700">Account</p>
        <h1 className="mt-1 text-2xl font-bold text-slate-950">Security</h1>
        <p className="mt-2 text-sm text-slate-600">Change your password and invalidate existing sessions.</p>
      </div>
      <AccountTabs />
      <form className="max-w-xl space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
        {submissionError ? <Alert>{submissionError}</Alert> : null}
        {[
          ["oldPassword", "Current password", "current-password"],
          ["newPassword", "New password", "new-password"],
          ["confirmPassword", "Confirm new password", "new-password"],
        ].map(([name, label, autoComplete]) => {
          const fieldName = name as keyof PasswordFields;
          return (
            <div key={name}>
              <label className="mb-1.5 block text-sm font-semibold text-slate-800" htmlFor={name}>
                {label}
              </label>
              <Input
                id={name}
                type="password"
                autoComplete={autoComplete}
                aria-invalid={Boolean(errors[fieldName])}
                aria-describedby={errors[fieldName] ? `${name}-error` : undefined}
                {...register(fieldName)}
              />
              <FieldError id={`${name}-error`} message={errors[fieldName]?.message} />
            </div>
          );
        })}
        <Button type="submit" isLoading={isSubmitting} icon={<KeyRound className="h-4 w-4" aria-hidden="true" />}>
          Update password
        </Button>
      </form>
    </section>
  );
};

export default PasswordPage;
