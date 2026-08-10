const FieldError = ({ id, message }: { id?: string; message?: string }) =>
  message ? (
    <p id={id} role="alert" className="mt-1.5 text-xs font-semibold text-red-600">
      {message}
    </p>
  ) : null;

export default FieldError;
