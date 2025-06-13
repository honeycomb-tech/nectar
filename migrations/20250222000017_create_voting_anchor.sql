CREATE TABLE voting_anchors (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    block_id BIGINT NOT NULL,
    data_hash VARBINARY(32) NOT NULL,
    url VARCHAR(128) NOT NULL,
    FOREIGN KEY (block_id) REFERENCES blocks (id)
);
