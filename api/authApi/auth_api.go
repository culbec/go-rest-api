package authapi

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/culbec/go-rest-api/db"
	"github.com/culbec/go-rest-api/types"
	"github.com/culbec/go-rest-api/utils/security"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

type AuthHandler struct {
	db             *db.Client
	hasher         *security.Argon2idHash
	jwtManager     *security.JWTManager
	tokenBlacklist map[string]time.Time
	blacklistMutex *sync.RWMutex
}

func NewAuthHandler(db *db.Client, secretKey []byte) *AuthHandler {
	hasher := security.NewArgon2idHash(
		security.DEFAULT_TIME,
		security.DEFAULT_MEMORY,
		security.DEFAULT_THREADS,
		uint32(len(secretKey)),
		security.DEFAULT_SALT_LEN,
	)
	jwtManager := security.NewJWTManager(secretKey, 60*60*60)

	return &AuthHandler{
		db:             db,
		hasher:         hasher,
		jwtManager:     jwtManager,
		tokenBlacklist: make(map[string]time.Time),
		blacklistMutex: &sync.RWMutex{},
	}
}

func (h *AuthHandler) ValidateToken(ctx *gin.Context) (string, error) {
	token := ctx.Query("token")
	if token == "" {
		return "", errors.New("token is missing")
	}

	username, err := h.jwtManager.ValidateToken(token)
	if err != nil {
		return "", errors.New("invalid token")
	}

	return username, nil
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	var req types.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user []types.User
	if status, err := h.db.QueryCollection("users", &bson.D{{Key: "username", Value: req.Username}}, nil, &user); err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	if len(user) == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	err := h.hasher.ComparePasswords(
		[]byte(req.Password),
		[]byte(user[0].Salt),
		[]byte(user[0].Password),
	)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := h.jwtManager.GenerateToken(req.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, types.AuthResponse{Token: token})
}

func (h *AuthHandler) Logout(ctx *gin.Context) {
	authHeader := ctx.GetHeader("Authorization")

	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid authorization header"})
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	h.blacklistMutex.Lock()
	h.tokenBlacklist[token] = time.Now()
	h.blacklistMutex.Unlock()

	ctx.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Register(ctx *gin.Context) {
	var req types.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashSalt, err := h.hasher.GenerateHash([]byte(req.Password), nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userDate := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	userVersion := 1

	user := types.User{
		Username: req.Username,
		Password: string(hashSalt.Hash),
		Salt:     string(hashSalt.Salt),
		Date:     userDate,
		Version:  userVersion,
	}

	// Inserting conditions for not adding the same user twice
	insertingConditions := &bson.D{
		{Key: "username", Value: user.Username},
	}

	id, status, err := h.db.InsertDocument("users", insertingConditions, user)
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Checking if no ID was provided by the server
	// In this case, the user already exists
	if id == nil {
		ctx.JSON(status, gin.H{"error": "User already exists"})
		return
	}

	// Generate token
	token, err := h.jwtManager.GenerateToken(req.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, types.AuthResponse{Token: token})
}
