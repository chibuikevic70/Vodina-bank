package auth

import (
  "context"
  "time"
  "os"
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/golang-jwt/jwt/v5"
  "github.com/google/uuid"
  "golang.org/x/crypto/bcrypt"
)

func Register(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    var req struct{ Email string `json:"email"`; Password string `json:"password"`; FullName string `json:"full_name"` }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":err.Error()}); return }
    hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
    id := uuid.New().String()
    _, err := pool.Exec(context.Background(), "INSERT INTO users (id, email, password_hash, full_name, created_at) VALUES ($1,$2,$3,$4,$5)", id, req.Email, string(hash), req.FullName, time.Now())
    if err != nil { c.JSON(400, gin.H{"error":"email exists or invalid"}); return }
    // auto create Naira wallet with 0 balance
    pool.Exec(context.Background(), "INSERT INTO wallets (id, user_id, type, name, currency, balance_kobo) VALUES ($1,$2,'NAIRA','Main Naira Wallet','NGN',0)", uuid.New().String(), id)
    c.JSON(201, gin.H{"user_id":id, "message":"Vodina account created. Now verify BVN/NIN for higher limits."})
  }
}

func Login(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    var req struct{ Email string `json:"email"`; Password string `json:"password"` }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":err.Error()}); return }
    var id, hash string
    err := pool.QueryRow(context.Background(), "SELECT id, password_hash FROM users WHERE email=$1", req.Email).Scan(&id,&hash)
    if err != nil { c.JSON(401, gin.H{"error":"invalid credentials"}); return }
    if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil { c.JSON(401, gin.H{"error":"invalid credentials"}); return }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id":id, "exp": time.Now().Add(24*time.Hour).Unix()})
    secret := os.Getenv("JWT_SECRET"); if secret=="" { secret="vodina-dev-secret" }
    t, _ := token.SignedString([]byte(secret))
    c.JSON(200, gin.H{"token":t, "user_id":id})
  }
}

func Middleware() gin.HandlerFunc {
  return func(c *gin.Context){
    tokenStr := c.GetHeader("Authorization")
    if len(tokenStr) > 7 && tokenStr[:7]=="Bearer " { tokenStr=tokenStr[7:] }
    secret := os.Getenv("JWT_SECRET"); if secret=="" { secret="vodina-dev-secret" }
    t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error){ return []byte(secret), nil })
    if err != nil || !t.Valid { c.JSON(401, gin.H{"error":"unauthorized - login first"}); c.Abort(); return }
    claims := t.Claims.(jwt.MapClaims)
    c.Set("user_id", claims["user_id"].(string))
    c.Next()
  }
}
