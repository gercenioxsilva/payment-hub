ALTER TABLE payment_intent
  ADD COLUMN IF NOT EXISTS card_brand TEXT,
  ADD COLUMN IF NOT EXISTS card_masked_pan TEXT,
  ADD COLUMN IF NOT EXISTS card_auth_code TEXT;
