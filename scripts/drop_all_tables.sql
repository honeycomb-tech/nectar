-- Drop all Nectar tables for fresh sync
-- WARNING: This will delete ALL data!

SET FOREIGN_KEY_CHECKS = 0;

-- UTXO Tracking Tables
DROP TABLE IF EXISTS utxo_deltas;
DROP TABLE IF EXISTS utxo_states;

-- Checkpoint Tables
DROP TABLE IF EXISTS checkpoint_resume_log;
DROP TABLE IF EXISTS sync_checkpoints;

-- Off-chain Tables
DROP TABLE IF EXISTS off_chain_vote_fetch_errors;
DROP TABLE IF EXISTS off_chain_vote_external_updates;
DROP TABLE IF EXISTS off_chain_vote_references;
DROP TABLE IF EXISTS off_chain_vote_authors;
DROP TABLE IF EXISTS off_chain_vote_d_rep_data;
DROP TABLE IF EXISTS off_chain_vote_gov_action_data;
DROP TABLE IF EXISTS off_chain_vote_data;
DROP TABLE IF EXISTS off_chain_pool_fetch_errors;
DROP TABLE IF EXISTS off_chain_pool_data;

-- Governance Tables
DROP TABLE IF EXISTS drep_infos;
DROP TABLE IF EXISTS instant_rewards;
DROP TABLE IF EXISTS pot_transfers;
DROP TABLE IF EXISTS reserves;
DROP TABLE IF EXISTS treasuries;
DROP TABLE IF EXISTS constitutions;
DROP TABLE IF EXISTS committees;
DROP TABLE IF EXISTS d_rep_distrs;
DROP TABLE IF EXISTS epoch_states;
DROP TABLE IF EXISTS committee_members;
DROP TABLE IF EXISTS treasury_withdrawals;
DROP TABLE IF EXISTS committee_deregistrations;
DROP TABLE IF EXISTS committee_registrations;
DROP TABLE IF EXISTS delegation_votes;
DROP TABLE IF EXISTS voting_anchors;
DROP TABLE IF EXISTS committee_hashes;
DROP TABLE IF EXISTS d_rep_hashes;
DROP TABLE IF EXISTS param_proposals;
DROP TABLE IF EXISTS gov_action_proposals;
DROP TABLE IF EXISTS voting_procedures;

-- Staking Tables
DROP TABLE IF EXISTS instant_rewards;
DROP TABLE IF EXISTS reserved_pool_tickers;
DROP TABLE IF EXISTS delisted_pools;
DROP TABLE IF EXISTS epoch_stake_progresses;
DROP TABLE IF EXISTS reward_rests;
DROP TABLE IF EXISTS pool_stats;
DROP TABLE IF EXISTS epoch_stakes;
DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS rewards;
DROP TABLE IF EXISTS delegations;
DROP TABLE IF EXISTS stake_deregistrations;
DROP TABLE IF EXISTS stake_registrations;
DROP TABLE IF EXISTS pool_metadata_refs;
DROP TABLE IF EXISTS pool_owners;
DROP TABLE IF EXISTS pool_retires;
DROP TABLE IF EXISTS pool_relays;
DROP TABLE IF EXISTS pool_updates;
DROP TABLE IF EXISTS pool_hashes;
DROP TABLE IF EXISTS stake_addresses;

-- Asset Tables
DROP TABLE IF EXISTS event_infos;
DROP TABLE IF EXISTS ada_pots;
DROP TABLE IF EXISTS epoch_params;
DROP TABLE IF EXISTS cost_models;
DROP TABLE IF EXISTS extra_key_witnesses;
DROP TABLE IF EXISTS tx_cbors;
DROP TABLE IF EXISTS tx_metadata;
DROP TABLE IF EXISTS collateral_tx_outs;
DROP TABLE IF EXISTS reference_tx_ins;
DROP TABLE IF EXISTS collateral_tx_ins;
DROP TABLE IF EXISTS redeemers;
DROP TABLE IF EXISTS redeemer_data;
DROP TABLE IF EXISTS data;
DROP TABLE IF EXISTS scripts;
DROP TABLE IF EXISTS ma_tx_mints;
DROP TABLE IF EXISTS ma_tx_outs;
DROP TABLE IF EXISTS multi_assets;

-- Core Tables
DROP TABLE IF EXISTS epoches;
DROP TABLE IF EXISTS slot_leaders;
DROP TABLE IF EXISTS tx_ins;
DROP TABLE IF EXISTS tx_outs;
DROP TABLE IF EXISTS txes;
DROP TABLE IF EXISTS blocks;

SET FOREIGN_KEY_CHECKS = 1;

-- Verify all tables are dropped
SELECT 'Tables remaining after drop:' AS status;
SELECT TABLE_NAME 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = DATABASE()
ORDER BY TABLE_NAME;