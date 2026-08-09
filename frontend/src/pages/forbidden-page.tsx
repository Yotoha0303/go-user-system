import { ShieldX } from "lucide-react";
import { Link } from "react-router-dom";
import { buttonClassName } from "../components/elements/button-styles";

const ForbiddenPage = () => (
  <section className="mx-auto max-w-xl py-16 text-center">
    <ShieldX className="mx-auto h-10 w-10 text-red-600" aria-hidden="true" />
    <p className="mt-5 text-sm font-semibold text-red-700">403</p>
    <h1 className="mt-1 text-2xl font-bold text-slate-950">Access denied</h1>
    <p className="mt-3 text-sm leading-6 text-slate-600">Your account does not have permission to open this page.</p>
    <Link to="/profile" className={buttonClassName({ className: "mt-6" })}>Return to profile</Link>
  </section>
);

export default ForbiddenPage;
