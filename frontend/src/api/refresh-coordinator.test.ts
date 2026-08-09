import { createRefreshCoordinator } from "./refresh-coordinator";

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
});
