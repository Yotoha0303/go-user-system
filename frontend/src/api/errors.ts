import axios from "axios";
import type { ApiResponse } from "./types";

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status?: number,
    readonly code?: number
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export const normalizeApiError = (
  error: unknown,
  fallback = "Request failed. Please try again."
): ApiError => {
  if (error instanceof ApiError) return error;

  if (axios.isAxiosError<ApiResponse<null>>(error)) {
    return new ApiError(
      error.response?.data?.msg || error.message || fallback,
      error.response?.status,
      error.response?.data?.code
    );
  }

  if (error instanceof Error) return new ApiError(error.message);
  return new ApiError(fallback);
};

export const errorMessage = (error: unknown, fallback?: string): string =>
  normalizeApiError(error, fallback).message;
