-- Vodina Bank - STEP 1-5 Schema
-- Currency in KOBO (1 NGN = 100 kobo) - never float!

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- USERS
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  phone TEXT,
  bvn TEXT UNIQUE,
  nin TEXT UNIQUE,
  kyc_status TEXT DEFAULT 'PENDING' CHECK (kyc_status IN ('PENDING','BVN_VERIFIED','NIN_VERIFIED','FULLY_VERIFIED','REJECTED')),
  kyc_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- WALLETS (Naira Wallet)
CREATE TABLE wallets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  type TEXT DEFAULT 'NAIRA' CHECK (type IN ('NAIRA','SAVINGS','LOAN_DISBURSEMENT')),
  name TEXT NOT NULL DEFAULT 'Main Naira Wallet',
  currency TEXT DEFAULT 'NGN',
  balance_kobo BIGINT NOT NULL DEFAULT 0 CHECK (balance_kobo >= 0),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- IDEMPOTENCY (prevent double debit)
CREATE TABLE idempotency_keys (
  key TEXT PRIMARY KEY,
  user_id UUID,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- TRANSACTIONS (header)
CREATE TABLE transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  from_wallet_id UUID REFERENCES wallets(id),
  to_wallet_id UUID REFERENCES wallets(id),
  to_bank_account TEXT,
  to_bank_code TEXT,
  amount_kobo BIGINT NOT NULL CHECK (amount_kobo > 0),
  fee_kobo BIGINT DEFAULT 0,
  type TEXT CHECK (type IN ('INTERNAL','NIP','SAVINGS_LOCK','SAVINGS_UNLOCK','LOAN_DISBURSE','LOAN_REPAY')),
  status TEXT DEFAULT 'PENDING' CHECK (status IN ('PENDING','SUCCESS','FAILED')),
  idempotency_key TEXT UNIQUE NOT NULL REFERENCES idempotency_keys(key),
  narration TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- LEDGER (immutable audit trail)
CREATE TABLE ledger_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  wallet_id UUID REFERENCES wallets(id),
  txn_id UUID REFERENCES transactions(id),
  entry_type TEXT CHECK (entry_type IN ('DEBIT','CREDIT')),
  amount_kobo BIGINT NOT NULL,
  balance_after_kobo BIGINT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ledger_wallet_time ON ledger_entries(wallet_id, created_at DESC);

-- SAVINGS (Step 3)
CREATE TABLE savings_goals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id),
  wallet_id UUID REFERENCES wallets(id), -- locked wallet
  name TEXT NOT NULL, -- e.g., Rent, School
  target_kobo BIGINT NOT NULL,
  saved_kobo BIGINT DEFAULT 0,
  interest_rate_bp INT DEFAULT 1500, -- 15% = 1500 basis points
  lock_until DATE,
  status TEXT DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','COMPLETED','BROKEN')),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- LOANS (Step 5)
CREATE TABLE loans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id),
  amount_requested_kobo BIGINT NOT NULL,
  amount_approved_kobo BIGINT,
  interest_rate_bp INT NOT NULL DEFAULT 3000, -- 30% annual
  tenure_days INT NOT NULL DEFAULT 30,
  status TEXT DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','DISBURSED','REPAID','DEFAULTED','REJECTED')),
  credit_score INT,
  disbursed_at TIMESTAMPTZ,
  due_date DATE,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE loan_repayments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  loan_id UUID REFERENCES loans(id),
  due_date DATE NOT NULL,
  amount_due_kobo BIGINT NOT NULL,
  amount_paid_kobo BIGINT DEFAULT 0,
  status TEXT DEFAULT 'PENDING' CHECK (status IN ('PENDING','PAID','LATE','MISSED')),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- KYC ATTEMPTS (Step 2)
CREATE TABLE kyc_verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id),
  type TEXT CHECK (type IN ('BVN','NIN')),
  id_number TEXT NOT NULL,
  provider TEXT DEFAULT 'MOCK', -- MOCK, SMILE_ID, YOUVERIFY
  status TEXT DEFAULT 'PENDING',
  response_json JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
