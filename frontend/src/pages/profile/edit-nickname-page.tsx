import { yupResolver } from "@hookform/resolvers/yup";
import { Save } from "lucide-react";
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
    <section className="mx-auto w-full max-w-xl">
      <p className="text-sm font-semibold text-blue-700">Profile</p>
      <h1 className="mt-1 text-2xl font-bold text-slate-950">Edit nickname</h1>
      <form className="mt-7 space-y-5 rounded-lg border border-slate-200 bg-white p-5 sm:p-6" onSubmit={handleSubmit(onSubmit)} noValidate>
        {submissionError ? <Alert>{submissionError}</Alert> : null}
        <div>
          <label className="mb-1.5 block text-sm font-semibold text-slate-800" htmlFor="nickname">
            Nickname
          </label>
          <Input
            id="nickname"
            autoComplete="nickname"
            aria-invalid={Boolean(errors.nickname)}
            aria-describedby={errors.nickname ? "nickname-error" : undefined}
            {...register("nickname")}
          />
          <FieldError id="nickname-error" message={errors.nickname?.message} />
        </div>
        <div className="flex flex-wrap gap-3">
          <Button type="submit" isLoading={isSubmitting} icon={<Save className="h-4 w-4" aria-hidden="true" />}>
            Save changes
          </Button>
          <Link to="/profile" className={buttonClassName({ variant: "secondary" })}>
            Cancel
          </Link>
        </div>
      </form>
    </section>
  );
};

export default EditNicknamePage;
