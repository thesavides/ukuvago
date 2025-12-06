package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	config *config.Config
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{config: cfg}
}

// JWT Claims
type Claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// HashPassword creates a bcrypt hash of the password
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a password with a hash
func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a JWT token for a user
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(time.Duration(s.config.JWTExpiration) * time.Hour)

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.AppName,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GenerateRandomToken generates a random token for email verification or password reset
func (s *AuthService) GenerateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Register creates a new user account
func (s *AuthService) Register(email, password, firstName, lastName string, role models.UserRole) (*models.User, error) {
	db := database.GetDB()

	// Check if user already exists
	var existingUser models.User
	if err := db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	passwordHash, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Generate verification token
	verifyToken, err := s.GenerateRandomToken()
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
		Role:         role,
		VerifyToken:  verifyToken,
	}

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and returns a token
func (s *AuthService) Login(email, password string) (*models.User, string, error) {
	db := database.GetDB()

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, "", errors.New("invalid credentials")
	}

	token, err := s.GenerateToken(&user)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}

// VerifyEmail verifies a user's email address
func (s *AuthService) VerifyEmail(token string) error {
	db := database.GetDB()

	var user models.User
	if err := db.Where("verify_token = ?", token).First(&user).Error; err != nil {
		return errors.New("invalid verification token")
	}

	user.EmailVerified = true
	user.VerifyToken = ""
	return db.Save(&user).Error
}

// InitiatePasswordReset creates a password reset token
func (s *AuthService) InitiatePasswordReset(email string) (string, error) {
	db := database.GetDB()

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		// Don't reveal if email exists
		return "", nil
	}

	token, err := s.GenerateRandomToken()
	if err != nil {
		return "", err
	}

	expires := time.Now().Add(24 * time.Hour)
	user.ResetToken = token
	user.ResetExpires = &expires

	if err := db.Save(&user).Error; err != nil {
		return "", err
	}

	return token, nil
}

// ResetPassword resets a user's password using a reset token
func (s *AuthService) ResetPassword(token, newPassword string) error {
	db := database.GetDB()

	var user models.User
	if err := db.Where("reset_token = ?", token).First(&user).Error; err != nil {
		return errors.New("invalid reset token")
	}

	if user.ResetExpires == nil || time.Now().After(*user.ResetExpires) {
		return errors.New("reset token has expired")
	}

	passwordHash, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = passwordHash
	user.ResetToken = ""
	user.ResetExpires = nil

	return db.Save(&user).Error
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(id uuid.UUID) (*models.User, error) {
	db := database.GetDB()

	var user models.User
	if err := db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by their email
func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	db := database.GetDB()

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates user profile information
func (s *AuthService) UpdateUser(user *models.User) error {
	db := database.GetDB()
	return db.Save(user).Error
}
