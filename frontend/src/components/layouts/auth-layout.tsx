import { ReactNode } from "react";

type AuthLayoutProps = {
  title: string;
  subtitle: string;
  children: ReactNode;
  footer: ReactNode;
};

const AuthLayout = ({ title, subtitle, children, footer }: AuthLayoutProps) => (
  <section className="mx-auto flex min-w-0 w-full max-w-sm flex-col py-10 sm:py-16">
    <div className="mb-7">
      <p className="mb-2 text-sm font-semibold text-blue-700">Go User System</p>
      <h1 className="text-2xl font-bold text-slate-950">{title}</h1>
      <p className="mt-2 text-sm leading-6 text-slate-600">{subtitle}</p>
    </div>
    <div className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm sm:p-6">
      {children}
    </div>
    <div className="mt-5 text-center text-sm text-slate-600">{footer}</div>
  </section>
);

export default AuthLayout;
