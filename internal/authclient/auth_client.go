package authclient

import (
	"context"
	"fmt"

	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/AuthContract/gen/authpb"
	"google.golang.org/grpc"
)

type Client struct {
	api authpb.AuthServiceClient
}

func New(conn *grpc.ClientConn) *Client {
	return &Client{
		api: authpb.NewAuthServiceClient(conn),
	}
}

func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	req := &authpb.LoginRequest{
		Login:    email,
		Password: password,
	}
	resp, err := c.api.Login(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return resp.Token, nil
}

func (c *Client) ResetPassword(ctx context.Context, email string) error {
	req := &authpb.ResetPasswordRequest{
		Email: email,
	}
	_, err := c.api.ResetPassword(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}
	return nil
}

func (c *Client) ValidateToken(ctx context.Context, token string) (valid bool, userID string, err error) {
	req := &authpb.ValidateTokenRequest{
		Token: token,
	}
	resp, err := c.api.ValidateToken(ctx, req)
	if err != nil {
		return false, "", fmt.Errorf("failed to validate token: %w", err)
	}
	return resp.Valid, resp.UserId, nil
}

func (c *Client) Register(ctx context.Context, req dto.RegisterRequest) (string, error) {
	r := &authpb.RegisterRequest{
		Email:     req.Email,
		FirstName: req.Name,
		LastName:  req.Surname,
		Phone:     req.PhoneNumber,
		Password:  req.Password,
	}
	resp, err := c.api.Register(ctx, r)
	if err != nil {
		return "", fmt.Errorf("failed to register new user: %w", err)
	}
	return resp.Token, nil
}
