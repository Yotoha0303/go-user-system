package response

type UserInfoResponse struct {
	ID       int64  `json:"id"`
	UserName string `json:"username"`
	NickName string `json:"nickname"`
	Status   int8   `json:"status"`
}
