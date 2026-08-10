import { logout } from "./auth.api";
import { publicApi } from "./client";

vi.mock("./client", () => ({
  publicApi: { post: vi.fn() },
}));

describe("logout", () => {
  afterEach(() => vi.clearAllMocks());

  it("presents the current access token without using the refresh interceptor", async () => {
    vi.mocked(publicApi.post).mockResolvedValue({
      data: { code: 0, msg: "success", data: null },
    });

    await logout("access-token");

    expect(publicApi.post).toHaveBeenCalledWith(
      "/api/v1/auth/logout",
      undefined,
      { headers: { Authorization: "Bearer access-token" } }
    );
  });
});
