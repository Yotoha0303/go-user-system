import { LoaderCircle } from "lucide-react";

const Spinner = ({ size = "md" }: { size?: "sm" | "md" }) => (
  <LoaderCircle
    aria-hidden="true"
    className={`${size === "sm" ? "h-4 w-4" : "h-5 w-5"} animate-spin`}
  />
);

export default Spinner;
