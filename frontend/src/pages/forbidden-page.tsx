import { ArrowLeft, ShieldX } from "lucide-react";
import { Link } from "react-router-dom";
import { buttonClassName } from "../components/elements/button-styles";

const ForbiddenPage = () => (
  <section className="mx-auto flex min-h-[60vh] w-full max-w-xl items-center justify-center py-10">
    <div className="surface-shadow w-full rounded-lg border border-slate-200 bg-white p-7 text-center sm:p-10">
      <span className="mx-auto grid h-12 w-12 place-items-center rounded-md bg-red-50 text-red-700">
        <ShieldX className="h-6 w-6" aria-hidden="true" />
      </span>
      <p className="mt-5 text-xs font-bold uppercase text-red-700">Error 403</p>
      <h1 className="mt-2 text-2xl font-bold text-slate-950">Access denied</h1>
      <p className="mx-auto mt-3 max-w-sm text-sm leading-6 text-slate-600">
        Your account does not have permission to open this page.
      </p>
      <Link to="/profile" className={buttonClassName({ className: "mt-6" })}>
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Return to profile
      </Link>
    </div>
  </section>
);

export default ForbiddenPage;
