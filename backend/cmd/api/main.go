package main

import (
  "context"
  "log"
  "os"
  "vodina-bank/internal/auth"
  "vodina-bank/internal/wallet"
  "vodina-bank/internal/ledger"
  "vodina-bank/internal/kyc"
  "vodina-bank/internal/savings"
  "vodina-bank/internal/loans"
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/joho/godotenv"
)

func main(){
  godotenv.Load()
  ctx := context.Background()
  dbUrl := os.Getenv("DATABASE_URL")
  if dbUrl == "" { dbUrl = "postgres://vodina:vodina@localhost:5432/vodina?sslmode=disable" }
  pool, err := pgxpool.New(ctx, dbUrl)
  if err != nil { log.Fatal(err) }

  r := gin.Default()
  r.GET("/", func(c *gin.Context){ c.JSON(200, gin.H{"bank":"Vodina Bank","status":"building step by step"}) })
  r.GET("/health", func(c *gin.Context){ c.JSON(200, gin.H{"ok":true}) })

  r.POST("/auth/register", auth.Register(pool))
  r.POST("/auth/login", auth.Login(pool))

  api := r.Group("/api", auth.Middleware())
  {
    // Wallet
    api.POST("/wallets", wallet.CreateWallet(pool))
    api.GET("/wallets", wallet.ListWallets(pool))
    api.GET("/wallets/:id/balance", wallet.GetBalance(pool))
    
    // Transfers
    api.POST("/transfers", ledger.Transfer(pool))
    api.GET("/transactions", ledger.ListTransactions(pool))

    // KYC - BVN/NIN
    api.POST("/kyc/bvn", kyc.VerifyBVN(pool))
    api.POST("/kyc/nin", kyc.VerifyNIN(pool))
    api.GET("/kyc/status", kyc.GetStatus(pool))

    // Savings
    api.POST("/savings", savings.CreateGoal(pool))
    api.GET("/savings", savings.ListGoals(pool))
    api.POST("/savings/:id/lock", savings.LockMoney(pool))

    // Loans
    api.POST("/loans/apply", loans.Apply(pool))
    api.GET("/loans", loans.List(pool))
    api.POST("/loans/:id/repay", loans.Repay(pool))
  }

  port := os.Getenv("PORT"); if port==""{port="8080"}
  log.Printf("Vodina Bank running on :%s", port)
  r.Run(":"+port)
}
