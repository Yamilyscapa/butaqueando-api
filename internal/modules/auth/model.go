package auth

type SignInRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type SignOutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type AuthUserData struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

type AuthTokensData struct {
	TokenType             string        `json:"tokenType"`
	AccessToken           string        `json:"accessToken"`
	RefreshToken          string        `json:"refreshToken"`
	AccessTokenExpiresIn  int64         `json:"accessTokenExpiresIn"`
	RefreshTokenExpiresIn int64         `json:"refreshTokenExpiresIn"`
	User                  *AuthUserData `json:"user,omitempty"`
}

type SignOutData struct {
	OK bool `json:"ok"`
}

type RefreshClaims struct {
	UserID  string
	Role    string
	TokenID string
}
