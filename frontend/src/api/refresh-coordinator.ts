export const createRefreshCoordinator = <T>(refresh: () => Promise<T>) => {
  let pending: Promise<T> | null = null;

  return () => {
    if (!pending) {
      pending = refresh().finally(() => {
        pending = null;
      });
    }
    return pending;
  };
};
