package dto

type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Surname     string `json:"surname" validate:"required,min=1,max=100"`
	PhoneNumber string `json:"phone_number" validate:"required,e164"`
	Password    string `json:"password" validate:"required,min=8,max=40"`
}

type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type TokenResponse struct {
	Token string `json:"token"`
}
