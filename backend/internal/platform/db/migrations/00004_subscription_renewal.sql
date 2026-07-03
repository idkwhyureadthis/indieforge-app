-- +goose Up

-- expires_at: when the current subscription period ends (NULL = legacy/lifetime)
-- payment_method_id: YooKassa saved method for auto-renewal (empty = no auto-renew)
ALTER TABLE subscriptions
    ADD COLUMN expires_at         TIMESTAMPTZ,
    ADD COLUMN payment_method_id  TEXT NOT NULL DEFAULT '';

-- sub_id links a renewal payment back to the subscription it is renewing
ALTER TABLE payments ADD COLUMN sub_id TEXT REFERENCES subscriptions(id);
ALTER TABLE payments ADD COLUMN payment_method_id TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_subscriptions_expires ON subscriptions(expires_at)
    WHERE active = TRUE AND expires_at IS NOT NULL;

CREATE INDEX idx_payments_sub ON payments(sub_id) WHERE sub_id IS NOT NULL;

-- +goose Down

ALTER TABLE payments DROP COLUMN IF EXISTS payment_method_id;
ALTER TABLE payments DROP COLUMN IF EXISTS sub_id;
ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS payment_method_id,
    DROP COLUMN IF EXISTS expires_at;
