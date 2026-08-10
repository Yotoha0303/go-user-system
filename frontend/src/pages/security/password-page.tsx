import { yupResolver } from "@hookform/resolvers/yup";
import { KeyRound, LogOut, ShieldCheck } from "lucide-react";
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
import PageHeader from "../../components/elements/page-header";

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
    <section className="w-full">
      <PageHeader
        eyebrow="Account"
        title="Security"
        description="Change your password and invalidate all existing sessions."
        icon={<ShieldCheck className="h-5 w-5" aria-hidden="true" />}
      />
      <AccountTabs />

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">
        <form
          className="surface-shadow overflow-hidden rounded-lg border border-slate-200 bg-white"
          onSubmit={handleSubmit(onSubmit)}
          noValidate
        >
          <div className="border-b border-slate-200 px-5 py-4 sm:px-6">
            <h2 className="text-sm font-bold text-slate-950">Update password</h2>
            <p className="mt-1 text-sm text-slate-500">
              Use a unique password between 12 and 72 characters.
            </p>
          </div>
          <div className="space-y-5 p-5 sm:p-6">
            {submissionError ? <Alert>{submissionError}</Alert> : null}
            {[
              ["oldPassword", "Current password", "current-password"],
              ["newPassword", "New password", "new-password"],
              ["confirmPassword", "Confirm new password", "new-password"],
            ].map(([name, label, autoComplete]) => {
              const fieldName = name as keyof PasswordFields;
              return (
                <div key={name}>
                  <label
                    className="mb-2 block text-sm font-semibold text-slate-800"
                    htmlFor={name}
                  >
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
                  <FieldError
                    id={`${name}-error`}
                    message={errors[fieldName]?.message}
                  />
                </div>
              );
            })}
          </div>
          <div className="border-t border-slate-200 bg-slate-50 px-5 py-4 sm:px-6">
            <Button
              type="submit"
              isLoading={isSubmitting}
              icon={<KeyRound className="h-4 w-4" aria-hidden="true" />}
            >
              Update password
            </Button>
          </div>
        </form>

        <aside className="surface-shadow rounded-lg border border-slate-200 bg-white p-5">
          <span className="grid h-9 w-9 place-items-center rounded-md bg-amber-50 text-amber-700">
            <LogOut className="h-4 w-4" aria-hidden="true" />
          </span>
          <h2 className="mt-4 text-sm font-bold text-slate-950">Session reset</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            After a successful update, all active sessions are revoked and you will
            return to the sign-in screen.
          </p>
        </aside>
      </div>
    </section>
  );
};

export default PasswordPage;
