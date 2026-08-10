import { AlertCircle, CheckCircle2 } from "lucide-react";

const Alert = ({
  children,
  tone = "error",
}: {
  children: string;
  tone?: "error" | "success";
}) => {
  const success = tone === "success";
  const Icon = success ? CheckCircle2 : AlertCircle;
  return (
    <div
      role={success ? "status" : "alert"}
      className={`flex items-start gap-2.5 rounded-md border px-3.5 py-3 text-sm font-medium ${
        success
          ? "border-teal-200 bg-teal-50 text-teal-900"
          : "border-red-200 bg-red-50 text-red-800"
      }`}
    >
      <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{children}</span>
    </div>
  );
};

export default Alert;
