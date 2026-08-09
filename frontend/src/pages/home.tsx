import { Navigate } from "react-router-dom";
import { selectAuthStatus } from "../app/authSlice";
import { useAppSelector } from "../app/hooks";

const HomePage = () => {
  const status = useAppSelector(selectAuthStatus);
  return <Navigate to={status === "authenticated" ? "/profile" : "/auth/login"} replace />;
};

export default HomePage;
