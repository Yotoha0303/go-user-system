package response

type UserInfoResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Status   int8   `json:"status"`
}

type TokenAndUserInfoResponse struct {
	AccessToken string           `json:"access_token"`
	User        UserInfoResponse `json:"user"`
}
