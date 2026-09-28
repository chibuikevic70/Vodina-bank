package kyc

import (
  "context"
  "time"
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/google/uuid"
)

// STEP 2 TEACHING: BVN = Bank Verification Number (11 digits, links all bank accounts)
// NIN = National Identification Number (11 digits, identity)
// CBN requires you verify at least one before allowing transactions > Tier 1 limits
// Real providers: Smile Identity, YouVerify, Dojah. We mock for learning.

func VerifyBVN(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    var req struct{ BVN string `json:"bvn" binding:"required"` }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":err.Error()}); return }
    if len(req.BVN)!=11 { c.JSON(400, gin.H{"error":"BVN must be 11 digits"}); return }

    // MOCK verification - in prod call Smile Identity API
    status := "VERIFIED"
    // Save verification
    _, err := pool.Exec(context.Background(), `INSERT INTO kyc_verifications (id, user_id, type, id_number, provider, status, response_json, created_at) VALUES ($1,$2,'BVN',$3,'MOCK',$4,'{"mock":true}',$5)`,
      uuid.New().String(), userID, req.BVN, status, time.Now())
    if err != nil { c.JSON(500, gin.H{"error":err.Error()}); return }

    // Update user
    pool.Exec(context.Background(), "UPDATE users SET bvn=$1, kyc_status='BVN_VERIFIED', kyc_verified_at=$2 WHERE id=$3", req.BVN, time.Now(), userID)

    c.JSON(200, gin.H{"status":status,"message":"BVN verified (mock). In prod, integrate Smile Identity."})
  }
}

func VerifyNIN(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    var req struct{ NIN string `json:"nin" binding:"required"` }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":err.Error()}); return }
    if len(req.NIN)!=11 { c.JSON(400, gin.H{"error":"NIN must be 11 digits"}); return }

    pool.Exec(context.Background(), `INSERT INTO kyc_verifications (id, user_id, type, id_number, provider, status, created_at) VALUES ($1,$2,'NIN',$3,'MOCK','VERIFIED',$4)`,
      uuid.New().String(), userID, req.NIN, time.Now())
    pool.Exec(context.Background(), "UPDATE users SET nin=$1, kyc_status='FULLY_VERIFIED' WHERE id=$2", req.NIN, userID)

    c.JSON(200, gin.H{"status":"VERIFIED","message":"NIN verified (mock)"})
  }
}

func GetStatus(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    var status, bvn, nin string
    pool.QueryRow(context.Background(), "SELECT kyc_status, COALESCE(bvn,''), COALESCE(nin,'') FROM users WHERE id=$1", userID).Scan(&status,&bvn,&nin)
    c.JSON(200, gin.H{"kyc_status":status,"bvn":bvn,"nin":nin,"tier": map[string]string{"PENDING":"Tier 0 - 50k limit","BVN_VERIFIED":"Tier 1 - 200k","FULLY_VERIFIED":"Tier 3 - Unlimited"}[status]})
  }
}
