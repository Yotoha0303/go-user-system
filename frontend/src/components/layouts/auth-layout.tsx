import { ReactNode } from "react";

type AuthLayoutProps = {
  title: string;
  subtitle: string;
  children: ReactNode;
  footer: ReactNode;
};

const AuthLayout = ({ title, subtitle, children, footer }: AuthLayoutProps) => (
  <section className="relative isolate min-h-[calc(100vh-4rem)] overflow-hidden bg-slate-100">
    <img
      className="absolute inset-0 h-full w-full object-cover object-left"
      src="/images/identity-access-background.webp"
      alt=""
    />
    <div className="absolute inset-0 bg-slate-950/10" aria-hidden="true" />
    <div className="relative mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-7xl items-center justify-end px-4 py-10 sm:px-6 lg:px-8">
      <div className="surface-shadow w-full max-w-md rounded-lg border border-white/80 bg-white/95 p-6 backdrop-blur-sm sm:p-8">
        <div className="mb-7">
          <p className="mb-2 text-xs font-bold uppercase text-teal-700">
            Identity workspace
          </p>
          <h1 className="text-2xl font-bold text-slate-950">{title}</h1>
          <p className="mt-2 text-sm leading-6 text-slate-600">{subtitle}</p>
        </div>
        {children}
        <div className="mt-6 border-t border-slate-200 pt-5 text-center text-sm text-slate-600">
          {footer}
        </div>
      </div>
    </div>
  </section>
);

export default AuthLayout;
