import { yupResolver } from "@hookform/resolvers/yup";
import { ArrowLeft, Pencil, Save } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";
import * as yup from "yup";
import { errorMessage } from "../../api/errors";
import { updateNickname } from "../../api/user.api";
import { profileUpdated, selectCurrentUser } from "../../app/authSlice";
import { useAppDispatch, useAppSelector } from "../../app/hooks";
import Alert from "../../components/elements/alert";
import Button from "../../components/elements/button";
import { buttonClassName } from "../../components/elements/button-styles";
import FieldError from "../../components/elements/field-error";
import Input from "../../components/elements/input";
import PageHeader from "../../components/elements/page-header";

type NicknameFields = { nickname: string };
const schema = yup
  .object({
    nickname: yup
      .string()
      .trim()
      .max(64, "Nickname must be at most 64 characters")
      .required("Nickname is required"),
  })
  .required();

const EditNicknamePage = () => {
  const user = useAppSelector(selectCurrentUser);
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [submissionError, setSubmissionError] = useState("");
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<NicknameFields>({
    resolver: yupResolver(schema),
    defaultValues: { nickname: user?.nickname ?? "" },
  });

  const onSubmit = async ({ nickname }: NicknameFields) => {
    setSubmissionError("");
    try {
      const normalized = nickname.trim();
      await updateNickname(normalized);
      if (user) dispatch(profileUpdated({ ...user, nickname: normalized }));
      navigate("/profile", {
        replace: true,
        state: { notice: "Nickname updated." },
      });
    } catch (error) {
      setSubmissionError(errorMessage(error, "Unable to update nickname."));
    }
  };

  return (
    <section className="w-full max-w-3xl">
      <PageHeader
        eyebrow="Profile"
        title="Edit nickname"
        description="Choose the display name used throughout your account workspace."
        icon={<Pencil className="h-5 w-5" aria-hidden="true" />}
        actions={
          <Link to="/profile" className={buttonClassName({ variant: "secondary" })}>
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            Back
          </Link>
        }
      />

      <form
        className="surface-shadow overflow-hidden rounded-lg border border-slate-200 bg-white"
        onSubmit={handleSubmit(onSubmit)}
        noValidate
      >
        <div className="border-b border-slate-200 px-5 py-4 sm:px-6">
          <h2 className="text-sm font-bold text-slate-950">Display information</h2>
          <p className="mt-1 text-sm text-slate-500">
            Your username stays unchanged: @{user?.username}
          </p>
        </div>
        <div className="space-y-5 p-5 sm:p-6">
          {submissionError ? <Alert>{submissionError}</Alert> : null}
          <div>
            <label
              className="mb-2 block text-sm font-semibold text-slate-800"
              htmlFor="nickname"
            >
              Nickname
            </label>
            <Input
              id="nickname"
              autoComplete="nickname"
              aria-invalid={Boolean(errors.nickname)}
              aria-describedby={errors.nickname ? "nickname-error" : "nickname-help"}
              {...register("nickname")}
            />
            <p id="nickname-help" className="mt-2 text-xs leading-5 text-slate-500">
              Use up to 64 characters.
            </p>
            <FieldError id="nickname-error" message={errors.nickname?.message} />
          </div>
        </div>
        <div className="flex flex-wrap gap-3 border-t border-slate-200 bg-slate-50 px-5 py-4 sm:px-6">
          <Button
            type="submit"
            isLoading={isSubmitting}
            icon={<Save className="h-4 w-4" aria-hidden="true" />}
          >
            Save changes
          </Button>
          <Link to="/profile" className={buttonClassName({ variant: "ghost" })}>
            Cancel
          </Link>
        </div>
      </form>
    </section>
  );
};

export default EditNicknamePage;
