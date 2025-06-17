-- Create table for required signers (Alonzo+ feature)
-- Stores the required signer key hashes for each transaction

CREATE TABLE required_signers (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tx_hash VARBINARY(32) NOT NULL,
    signer_index INT UNSIGNED NOT NULL,
    signer_hash VARBINARY(28) NOT NULL,
    UNIQUE KEY idx_required_signers_unique (tx_hash, signer_index),
    KEY idx_required_signers_tx (tx_hash),
    KEY idx_required_signers_hash (signer_hash)
);