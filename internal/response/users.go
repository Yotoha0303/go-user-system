package response

import "time"

type UserInfoResponse struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	Status      int8       `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type TokenAndUserInfoResponse struct {
	AccessToken           string           `json:"access_token"`
	AccessTokenExpiresIn  int64            `json:"access_token_expires_in"`
	RefreshTokenExpiresIn int64            `json:"refresh_token_expires_in"`
	User                  UserInfoResponse `json:"user"`
}

type TokenPairResponse struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresIn  int64  `json:"access_token_expires_in"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
}

type AuthorizationInfoResponse struct {
	RoleCodes       []string `json:"role_codes"`
	PermissionCodes []string `json:"permission_codes"`
}

type RoleResponse struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type PermissionResponse struct {
	ID     int64  `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Method string `json:"method"`
	Path   string `json:"path"`
}
