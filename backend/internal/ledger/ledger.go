package ledger

import (
  "context"
  "time"
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/google/uuid"
)

type TransferReq struct {
  IdempotencyKey string `json:"idempotency_key" binding:"required"`
  FromWalletID   string `json:"from_wallet_id" binding:"required"`
  ToWalletID     string `json:"to_wallet_id"`
  ToBankAccount  string `json:"to_bank_account"`
  ToBankCode     string `json:"to_bank_code"`
  AmountKobo     int64  `json:"amount_kobo" binding:"required"`
  Narration      string `json:"narration"`
}

func Transfer(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    var req TransferReq
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":err.Error()}); return }
    if req.AmountKobo <=0 { c.JSON(400, gin.H{"error":"amount must be >0"}); return }
    userID := c.GetString("user_id")
    ctx := context.Background()

    var exists bool
    pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE key=$1)", req.IdempotencyKey).Scan(&exists)
    if exists { c.JSON(200, gin.H{"status":"already_processed","key":req.IdempotencyKey}); return }

    tx, _ := pool.Begin(ctx)
    defer tx.Rollback(ctx)

    var fromBal int64
    err := tx.QueryRow(ctx, "SELECT balance_kobo FROM wallets WHERE id=$1 AND user_id=$2 FOR UPDATE", req.FromWalletID, userID).Scan(&fromBal)
    if err != nil { c.JSON(404, gin.H{"error":"from wallet not found"}); return }
    if fromBal < req.AmountKobo { c.JSON(400, gin.H{"error":"insufficient funds"}); return }

    txnID := uuid.New().String()
    tx.Exec(ctx, "UPDATE wallets SET balance_kobo = balance_kobo - $1 WHERE id=$2", req.AmountKobo, req.FromWalletID)
    if req.ToWalletID != "" {
      tx.Exec(ctx, "UPDATE wallets SET balance_kobo = balance_kobo + $1 WHERE id=$2", req.AmountKobo, req.ToWalletID)
    }

    tx.Exec(ctx, "INSERT INTO idempotency_keys (key, user_id) VALUES ($1,$2)", req.IdempotencyKey, userID)
    tx.Exec(ctx, "INSERT INTO transactions (id, from_wallet_id, to_wallet_id, to_bank_account, to_bank_code, amount_kobo, type, status, idempotency_key, narration) VALUES ($1,$2,$3,$4,$5,$6,$7,'SUCCESS',$8,$9)",
      txnID, req.FromWalletID, req.ToWalletID, req.ToBankAccount, req.ToBankCode, req.AmountKobo, map[bool]string{true:"INTERNAL",false:"NIP"}[req.ToWalletID!=""], req.IdempotencyKey, req.Narration)

    var toBal int64
    if req.ToWalletID != "" { tx.QueryRow(ctx, "SELECT balance_kobo FROM wallets WHERE id=$1", req.ToWalletID).Scan(&toBal) }

    tx.Exec(ctx, "INSERT INTO ledger_entries (id, wallet_id, txn_id, entry_type, amount_kobo, balance_after_kobo) VALUES ($1,$2,$3,'DEBIT',$4,$5)",
      uuid.New().String(), req.FromWalletID, txnID, req.AmountKobo, fromBal-req.AmountKobo)
    if req.ToWalletID != "" {
      tx.Exec(ctx, "INSERT INTO ledger_entries (id, wallet_id, txn_id, entry_type, amount_kobo, balance_after_kobo) VALUES ($1,$2,$3,'CREDIT',$4,$5)",
        uuid.New().String(), req.ToWalletID, txnID, req.AmountKobo, toBal)
    }

    tx.Commit(ctx)
    c.JSON(201, gin.H{"transaction_id":txnID,"status":"SUCCESS","amount_kobo":req.AmountKobo})
  }
}

func ListTransactions(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    rows, _ := pool.Query(context.Background(), "SELECT t.id, t.amount_kobo, t.status, t.created_at FROM transactions t JOIN wallets w ON w.id=t.from_wallet_id WHERE w.user_id=$1 ORDER BY t.created_at DESC LIMIT 50", userID)
    defer rows.Close()
    var list []gin.H
    for rows.Next(){
      var id string; var amt int64; var status string; var created time.Time
      rows.Scan(&id,&amt,&status,&created)
      list=append(list, gin.H{"id":id,"amount_kobo":amt,"status":status,"created_at":created})
    }
    c.JSON(200, list)
  }
}
