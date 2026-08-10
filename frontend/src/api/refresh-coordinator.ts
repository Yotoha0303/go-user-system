export type RefreshLockManager = {
  request: <T>(name: string, callback: () => Promise<T>) => Promise<T>;
};

type RefreshCoordinatorOptions = {
  lockManager?: RefreshLockManager | null;
  lockName?: string;
};

const browserLockManager = (): RefreshLockManager | null => {
  if (typeof navigator === "undefined" || !("locks" in navigator)) return null;
  return navigator.locks as RefreshLockManager;
};

export const createRefreshCoordinator = <T>(
  refresh: () => Promise<T>,
  options: RefreshCoordinatorOptions = {}
) => {
  let pending: Promise<T> | null = null;
  const lockManager =
    options.lockManager === undefined ? browserLockManager() : options.lockManager;
  const lockName = options.lockName ?? "go-user-system:refresh-token";

  return () => {
    if (!pending) {
      const coordinatedRefresh = lockManager
        ? lockManager.request(lockName, refresh)
        : refresh();
      pending = coordinatedRefresh.finally(() => {
        pending = null;
      });
    }
    return pending;
  };
};
