import {
  createRefreshCoordinator,
  type RefreshLockManager,
} from "./refresh-coordinator";

class SerialLockManager implements RefreshLockManager {
  private tail: Promise<unknown> = Promise.resolve();

  request<T>(_name: string, callback: () => Promise<T>): Promise<T> {
    const result = this.tail.then(callback);
    this.tail = result.then(
      () => undefined,
      () => undefined
    );
    return result;
  }
}

describe("createRefreshCoordinator", () => {
  it("shares one refresh request across concurrent callers", async () => {
    let resolveRefresh: ((value: string) => void) | undefined;
    const refresh = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          resolveRefresh = resolve;
        })
    );
    const coordinatedRefresh = createRefreshCoordinator(refresh);

    const first = coordinatedRefresh();
    const second = coordinatedRefresh();
    expect(refresh).toHaveBeenCalledTimes(1);

    resolveRefresh?.("new-access-token");
    await expect(first).resolves.toBe("new-access-token");
    await expect(second).resolves.toBe("new-access-token");

    refresh.mockImplementationOnce(() => Promise.resolve("next-access-token"));
    await coordinatedRefresh();
    expect(refresh).toHaveBeenCalledTimes(2);
  });

  it("serializes refresh requests across independent tab coordinators", async () => {
    const locks = new SerialLockManager();
    let resolveFirst: ((value: string) => void) | undefined;
    const firstRefresh = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          resolveFirst = resolve;
        })
    );
    const secondRefresh = vi.fn(() => Promise.resolve("second-token"));
    const firstTab = createRefreshCoordinator(firstRefresh, { lockManager: locks });
    const secondTab = createRefreshCoordinator(secondRefresh, { lockManager: locks });

    const first = firstTab();
    const second = secondTab();
    await Promise.resolve();

    expect(firstRefresh).toHaveBeenCalledTimes(1);
    expect(secondRefresh).not.toHaveBeenCalled();

    resolveFirst?.("first-token");
    await expect(first).resolves.toBe("first-token");
    await expect(second).resolves.toBe("second-token");
    expect(secondRefresh).toHaveBeenCalledTimes(1);
  });
});
