import { Outlet } from "react-router-dom";
import Navbar from "../header/navbar";

const MainLayout = () => (
  <div className="min-h-screen bg-slate-50">
    <Navbar />
    <main className="mx-auto min-w-0 w-full max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <Outlet />
    </main>
  </div>
);

export default MainLayout;
