import { ButtonHTMLAttributes, ReactNode } from "react";
import Spinner from "./spinner";
import { buttonClassName, ButtonSize, ButtonVariant } from "./button-styles";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  isLoading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: ReactNode;
};

const Button = ({
  children,
  className,
  disabled,
  icon,
  isLoading = false,
  size = "md",
  variant = "primary",
  ...props
}: ButtonProps) => (
  <button
    className={buttonClassName({ variant, size, className })}
    disabled={disabled || isLoading}
    aria-busy={isLoading}
    {...props}
  >
    {isLoading ? <Spinner size="sm" /> : icon}
    {children}
  </button>
);

export default Button;
