package service

import (
	"AuthService/internal/domain"
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo  domain.UserRepository
	tokenRepo domain.RefreshTokenRepository
	tokenSvc  *tokenService
}

func NewAuthService(
	userRepo domain.UserRepository,
	tokenRepo domain.RefreshTokenRepository,
	tokenSvc *tokenService,
) domain.AuthService {
	return &authService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		tokenSvc:  tokenSvc,
	}
}

func (service *authService) Register(ctx context.Context, input domain.RegisterInput) (*domain.TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:    input.Email,
		Password: string(hash),
	}
	if err := service.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return service.issueTokenPair(ctx, user)
}

func (service *authService) Login(ctx context.Context, input domain.LoginInput) (*domain.TokenPair, error) {
	user, err := service.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return service.issueTokenPair(ctx, user)
}

func (service *authService) Refresh(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	hash := HashToken(refreshToken)
	storedToken, err := service.tokenRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if time.Now().After(storedToken.ExpiresAt) {
		_ = service.tokenRepo.DeleteByHash(ctx, hash)
		return nil, domain.ErrTokenExpired
	}

	if err := service.tokenRepo.DeleteByHash(ctx, hash); err != nil {
		return nil, fmt.Errorf("delete old refresh token: %w", err)
	}

	user, err := service.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	return service.issueTokenPair(ctx, user)
}

func (service *authService) Logout(ctx context.Context, refreshToken string) error {
	hash := HashToken(refreshToken)
	return service.tokenRepo.DeleteByHash(ctx, hash)
}

func (service *authService) ValidateAccessToken(tokenStr string) (*domain.Claims, error) {
	return service.tokenSvc.ValidateAccessToken(tokenStr)
}

func (service *authService) issueTokenPair(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	accessToken, err := service.tokenSvc.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	rawRefresh, refreshHash, expiresAt, err := service.tokenSvc.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := service.tokenRepo.Create(ctx, &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}
	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}
