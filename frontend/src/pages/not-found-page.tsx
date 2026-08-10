import { ArrowLeft, SearchX } from "lucide-react";
import { Link } from "react-router-dom";
import { buttonClassName } from "../components/elements/button-styles";

const NotFoundPage = () => (
  <section className="mx-auto flex min-h-[60vh] w-full max-w-xl items-center justify-center py-10">
    <div className="surface-shadow w-full rounded-lg border border-slate-200 bg-white p-7 text-center sm:p-10">
      <span className="mx-auto grid h-12 w-12 place-items-center rounded-md bg-slate-100 text-slate-700">
        <SearchX className="h-6 w-6" aria-hidden="true" />
      </span>
      <p className="mt-5 text-xs font-bold uppercase text-slate-500">Error 404</p>
      <h1 className="mt-2 text-2xl font-bold text-slate-950">Page not found</h1>
      <p className="mx-auto mt-3 max-w-sm text-sm leading-6 text-slate-600">
        The requested page does not exist or has moved.
      </p>
      <Link to="/" className={buttonClassName({ className: "mt-6" })}>
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Go to account
      </Link>
    </div>
  </section>
);

export default NotFoundPage;
