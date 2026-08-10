import { Outlet } from "react-router-dom";
import { selectAuthStatus } from "../../app/authSlice";
import { useAppSelector } from "../../app/hooks";
import Navbar from "../header/navbar";

const MainLayout = () => {
  const status = useAppSelector(selectAuthStatus);
  const authenticated = status === "authenticated";

  return (
    <div className={authenticated ? "min-h-screen bg-[#f4f7f6]" : "min-h-screen bg-white"}>
      <Navbar />
      <div className={authenticated ? "lg:pl-64" : ""}>
        <main
          className={
            authenticated
              ? "mx-auto min-w-0 w-full max-w-7xl px-4 py-6 sm:px-6 sm:py-8 lg:px-8"
              : "min-w-0 w-full"
          }
        >
          <Outlet />
        </main>
      </div>
    </div>
  );
};

export default MainLayout;
