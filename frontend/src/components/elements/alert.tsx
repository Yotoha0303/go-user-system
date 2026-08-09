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
      className={`flex items-start gap-2 rounded-md border px-3 py-2 text-sm font-medium ${
        success
          ? "border-emerald-200 bg-emerald-50 text-emerald-800"
          : "border-red-200 bg-red-50 text-red-700"
      }`}
    >
      <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{children}</span>
    </div>
  );
};

export default Alert;
