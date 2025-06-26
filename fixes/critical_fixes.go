package fixes

// Critical fixes needed:

// 1. Fix slot_leaders slow query
// Change from:
//   tx.Clauses(clause.OnConflict{
//     Columns:   []clause.Column{{Name: "hash"}},
//     DoNothing: true,
//   }).Create(slotLeader)
//
// To batch inserts with raw SQL:
//   INSERT IGNORE INTO slot_leaders (hash, pool_hash, description) VALUES (?, ?, ?)

// 2. Fix StateQuery initialization
// Need to ensure:
// - StateQuery service actually starts
// - Epoch boundaries are detected
// - Rewards calculation is triggered

// 3. Fix connection pool exhaustion
// - Implement connection health checks
// - Add automatic reconnection
// - Better error handling for broken connections

// 4. Fix missing scripts
// - Check if script extraction is enabled
// - Verify script processor is being called

// 5. Fix index creation
// - Remove references to non-existent tables
// - Fix column names
// - Quote reserved words like `key`