const FieldError = ({ id, message }: { id?: string; message?: string }) =>
  message ? (
    <p id={id} role="alert" className="mt-1 text-sm font-medium text-red-600">
      {message}
    </p>
  ) : null;

export default FieldError;
