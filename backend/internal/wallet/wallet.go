package wallet

import (
  "context"
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/google/uuid"
  "time"
)

func CreateWallet(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    var req struct{ Name string `json:"name"` }
    if err := c.ShouldBindJSON(&req); err != nil { req.Name="Main Naira Wallet" }
    id := uuid.New().String()
    _, err := pool.Exec(context.Background(), "INSERT INTO wallets (id, user_id, type, name, currency, balance_kobo, created_at) VALUES ($1,$2,'NAIRA',$3,'NGN',0,$4)", id, userID, req.Name, time.Now())
    if err != nil { c.JSON(500, gin.H{"error":err.Error()}); return }
    c.JSON(201, gin.H{"wallet_id":id,"name":req.Name,"balance_kobo":0})
  }
}

func ListWallets(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    rows, _ := pool.Query(context.Background(), "SELECT id, name, type, balance_kobo FROM wallets WHERE user_id=$1", userID)
    defer rows.Close()
    var wallets []gin.H
    for rows.Next(){
      var id,name,typ string; var bal int64
      rows.Scan(&id,&name,&typ,&bal)
      wallets=append(wallets, gin.H{"id":id,"name":name,"type":typ,"balance_kobo":bal,"balance_naira":float64(bal)/100})
    }
    c.JSON(200, wallets)
  }
}

func GetBalance(pool *pgxpool.Pool) gin.HandlerFunc {
  return func(c *gin.Context){
    userID := c.GetString("user_id")
    wid := c.Param("id")
    var bal int64; var name string
    err := pool.QueryRow(context.Background(), "SELECT balance_kobo, name FROM wallets WHERE id=$1 AND user_id=$2", wid, userID).Scan(&bal,&name)
    if err != nil { c.JSON(404, gin.H{"error":"wallet not found"}); return }
    c.JSON(200, gin.H{"wallet_id":wid,"name":name,"balance_kobo":bal,"balance_naira":float64(bal)/100})
  }
}
