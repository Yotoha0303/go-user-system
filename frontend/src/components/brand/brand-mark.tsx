import { ShieldCheck } from "lucide-react";

type BrandMarkProps = {
  compact?: boolean;
  inverse?: boolean;
};

const BrandMark = ({ compact = false, inverse = false }: BrandMarkProps) => (
  <span className="inline-flex min-w-0 items-center gap-3">
    <span
      className={`grid h-9 w-9 shrink-0 place-items-center rounded-md ${
        inverse ? "bg-white text-slate-950" : "bg-slate-950 text-white"
      }`}
      aria-hidden="true"
    >
      <ShieldCheck className="h-5 w-5" strokeWidth={2} />
    </span>
    {!compact ? (
      <span className="min-w-0">
        <span
          className={`block truncate text-sm font-bold ${
            inverse ? "text-white" : "text-slate-950"
          }`}
        >
          Go User System
        </span>
        <span
          className={`block truncate text-xs ${
            inverse ? "text-slate-400" : "text-slate-500"
          }`}
        >
          Identity workspace
        </span>
      </span>
    ) : null}
  </span>
);

export default BrandMark;
