package response

var (
	CodeSuccess = 0

	CodeInvalidParams = 1001

	CodeUsernameAlreadyExists    = 2001
	CodeRegisterFailed           = 2002
	CodeUserNotFound             = 2003
	CodeUserDisabled             = 2004
	CodeLoginFailed              = 2005
	CodeUserPasswordNoDifference = 2006
	CodeUpdateUserPasswordFailed = 2007

	CodeTokenGenerateFailed   = 3001
	CodeTokenUserMissing      = 3002
	CodeTokenUserInvalid      = 3003
	CodeGetProfileFailed      = 3004
	CodeTokenMissing          = 3005
	CodeTokenInvalidFormat    = 3006
	CodeTokenExpired          = 3007
	CodeTokenInvalid          = 3008
	CodeTokenMalformed        = 3009
	CodeTokenSignatureInvalid = 3010

	CodeNicknameInvalid      = 4001
	CodeUpdateNicknameFailed = 4002

	CodeReadinessFailed = 5001

	CodeDatabaseNotInitialized = 5002

	CodeInternalError  = 6001
	CodeRequestTimeout = 6002
)
