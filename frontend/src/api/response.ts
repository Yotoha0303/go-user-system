import { ApiError } from "./errors";
import type { ApiResponse } from "./types";

export const unwrap = <T>(response: ApiResponse<T>): T => {
  if (response.code !== 0) {
    throw new ApiError(response.msg || "Request failed.", undefined, response.code);
  }
  return response.data;
};
