export type ApiResponse<T> = {
  code: number;
  msg: string;
  data: T;
};

export type UserProfile = {
  id: number;
  username: string;
  nickname: string;
  status: number;
  last_login_at?: string;
};

export type AuthorizationInfo = {
  role_codes: string[];
  permission_codes: string[];
};

export type AccessSession = {
  access_token: string;
  access_token_expires_in: number;
  refresh_token_expires_in: number;
};

export type LoginSession = AccessSession & {
  user: UserProfile;
};

export type Role = {
  id: number;
  code: string;
  name: string;
};

export type Permission = {
  id: number;
  code: string;
  name: string;
  method: string;
  path: string;
};

export const ApiCode = {
  success: 0,
  tokenExpired: 3007,
  permissionDenied: 5003,
} as const;

export const PermissionCode = {
  profileRead: "profile:read",
  profileUpdate: "profile:update",
  passwordUpdate: "password:update",
  adminRolesRead: "admin:roles:read",
  adminPermissionsRead: "admin:permissions:read",
  adminUserRolesUpdate: "admin:user_roles:update",
} as const;
