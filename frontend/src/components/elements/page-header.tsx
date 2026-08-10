import { ReactNode } from "react";

type PageHeaderProps = {
  eyebrow: string;
  title: string;
  description: string;
  icon?: ReactNode;
  actions?: ReactNode;
};

const PageHeader = ({
  actions,
  description,
  eyebrow,
  icon,
  title,
}: PageHeaderProps) => (
  <header className="mb-7 flex flex-wrap items-start justify-between gap-4">
    <div className="flex min-w-0 items-start gap-3">
      {icon ? (
        <span className="grid h-10 w-10 shrink-0 place-items-center rounded-md border border-teal-100 bg-teal-50 text-teal-800">
          {icon}
        </span>
      ) : null}
      <div className="min-w-0">
        <p className="text-xs font-bold uppercase text-teal-700">{eyebrow}</p>
        <h1 className="mt-1 text-2xl font-bold text-slate-950">{title}</h1>
        <p className="mt-1.5 max-w-2xl text-sm leading-6 text-slate-600">
          {description}
        </p>
      </div>
    </div>
    {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
  </header>
);

export default PageHeader;
