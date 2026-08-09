import { SearchX } from "lucide-react";
import { Link } from "react-router-dom";
import { buttonClassName } from "../components/elements/button-styles";

const NotFoundPage = () => (
  <section className="mx-auto max-w-xl py-16 text-center">
    <SearchX className="mx-auto h-10 w-10 text-slate-500" aria-hidden="true" />
    <p className="mt-5 text-sm font-semibold text-slate-500">404</p>
    <h1 className="mt-1 text-2xl font-bold text-slate-950">Page not found</h1>
    <p className="mt-3 text-sm leading-6 text-slate-600">The requested page does not exist or has moved.</p>
    <Link to="/" className={buttonClassName({ className: "mt-6" })}>Go to account</Link>
  </section>
);

export default NotFoundPage;
