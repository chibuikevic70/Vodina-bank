package loans

import (
  "context"
  "time"
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/google/uuid"
)

// STEP 5 TEACHING: Loans
// Flow: Apply -> Credit Check (BVN + transaction history) -> Approve -> Disburse (credit Naira wallet) -> Repayment schedule
// Interest in basis points, tenure in days. Repayment = principal + interest spread daily.

func Apply(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    var req struct{ AmountKobo int64 `json:"amount_kobo"`; TenureDays int `json:"tenure_days"` }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":err.Error()}); return }

    // Simple credit check: if KYC FULLY_VERIFIED and wallet balance > 0, approve 50%
    var kyc string; var bal int64
    pool.QueryRow(context.Background(), "SELECT kyc_status FROM users WHERE id=$1", userID).Scan(&kyc)
    pool.QueryRow(context.Background(), "SELECT COALESCE(SUM(balance_kobo),0) FROM wallets WHERE user_id=$1", userID).Scan(&bal)

    if kyc != "FULLY_VERIFIED" { c.JSON(400, gin.H{"error":"Complete BVN+NIN verification to access loans"}); return }

    status := "PENDING"
    approved := req.AmountKobo / 2 // mock: approve 50%
    creditScore := 650
    if bal > 5000000 { approved = req.AmountKobo; creditScore=750 } // 50k NGN balance

    loanID := uuid.New().String()
    due := time.Now().AddDate(0,0,req.TenureDays)

    _, err := pool.Exec(context.Background(), `INSERT INTO loans (id, user_id, amount_requested_kobo, amount_approved_kobo, interest_rate_bp, tenure_days, status, credit_score, due_date) VALUES ($1,$2,$3,$4,3000,$5,$6,$7,$8)`,
      loanID, userID, req.AmountKobo, approved, req.TenureDays, status, creditScore, due)
    if err != nil { c.JSON(500, gin.H{"error":err.Error()}); return }

    // Create repayment schedule
    daily := approved / int64(req.TenureDays)
    for i:=0; i<req.TenureDays; i++ {
      d := time.Now().AddDate(0,0,i+1)
      pool.Exec(context.Background(), "INSERT INTO loan_repayments (id, loan_id, due_date, amount_due_kobo) VALUES ($1,$2,$3,$4)", uuid.New().String(), loanID, d, daily)
    }

    c.JSON(201, gin.H{"loan_id":loanID,"requested_kobo":req.AmountKobo,"approved_kobo":approved,"interest_rate":"30% annual","tenure_days":req.TenureDays,"credit_score":creditScore,"status":status,"message":"Loan under review. Auto-disburse to Naira wallet when approved."})
  }
}

func List(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    rows, _ := pool.Query(context.Background(), "SELECT id, amount_approved_kobo, status, due_date FROM loans WHERE user_id=$1 ORDER BY created_at DESC", userID)
    defer rows.Close()
    var list []gin.H
    for rows.Next(){
      var id,status string; var amt int64; var due time.Time
      rows.Scan(&id,&amt,&status,&due)
      list=append(list, gin.H{"id":id,"amount_kobo":amt,"status":status,"due_date":due})
    }
    c.JSON(200, list)
  }
}

func Repay(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    c.JSON(200, gin.H{"message":"Repay logic: debit Naira wallet -> credit loan repayment. We build this after disbursement in Step 5."})
  }
}
