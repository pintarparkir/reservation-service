ALTER TABLE reservation
    ADD COLUMN payment_expires_at TIMESTAMPTZ;

CREATE INDEX idx_reservation_pending_payment_expiry
    ON reservation (payment_expires_at)
    WHERE state = 'PENDING_PAYMENT';
