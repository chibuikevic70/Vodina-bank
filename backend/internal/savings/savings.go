package savings

import (
  "context"
  "time"
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/google/uuid"
)

// STEP 3 TEACHING: Savings is a locked wallet. Money moves from main wallet to savings wallet.
// Interest accrues daily: interest = balance * rate / 365. We store in basis points (bp) to avoid float.
// 15% = 1500 bp. daily = 1500/36500 = 0.041% per day.

func CreateGoal(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    var req struct{
      Name string `json:"name" binding:"required"`
      TargetKobo int64 `json:"target_kobo" binding:"required"`
      LockUntil string `json:"lock_until"` // YYYY-MM-DD
    }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":err.Error()}); return }

    // Create locked savings wallet
    savingsWalletID := uuid.New().String()
    pool.Exec(context.Background(), "INSERT INTO wallets (id, user_id, type, name, currency, balance_kobo) VALUES ($1,$2,'SAVINGS',$3,'NGN',0)", savingsWalletID, userID, req.Name+" Savings")

    goalID := uuid.New().String()
    var lockDate *time.Time
    if req.LockUntil != "" { t,_ := time.Parse("2006-01-02", req.LockUntil); lockDate=&t }

    _, err := pool.Exec(context.Background(), "INSERT INTO savings_goals (id, user_id, wallet_id, name, target_kobo, interest_rate_bp, lock_until) VALUES ($1,$2,$3,$4,$5,1500,$6)",
      goalID, userID, savingsWalletID, req.Name, req.TargetKobo, lockDate)
    if err != nil { c.JSON(500, gin.H{"error":err.Error()}); return }

    c.JSON(201, gin.H{"savings_goal_id":goalID,"wallet_id":savingsWalletID,"message":"Savings goal created. Now lock money with POST /savings/:id/lock"})
  }
}

func ListGoals(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    rows, _ := pool.Query(context.Background(), "SELECT id, name, target_kobo, saved_kobo, status FROM savings_goals WHERE user_id=$1", userID)
    defer rows.Close()
    var list []gin.H
    for rows.Next(){
      var id,name,status string; var target,saved int64
      rows.Scan(&id,&name,&target,&saved,&status)
      list=append(list, gin.H{"id":id,"name":name,"target_kobo":target,"saved_kobo":saved,"progress":float64(saved)/float64(target)*100,"status":status})
    }
    c.JSON(200, list)
  }
}

func LockMoney(pool *pgxpool.Pool) gin.HandlerFunc {
  // Move money from main wallet to savings wallet - uses same ledger logic
  return func(c *gin.Context){
    c.JSON(200, gin.H{"message":"This calls same ledger.Transfer with from=main wallet to=savings wallet. We will wire it in Step 3 lab."})
  }
}
