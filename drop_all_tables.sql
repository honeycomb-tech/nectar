-- Drop all tables in nectar database for hash-based schema
-- This ensures a completely clean start for the new schema

SET FOREIGN_KEY_CHECKS = 0;

-- Drop all tables (order adjusted for foreign key dependencies)
-- Off-chain data tables
DROP TABLE IF EXISTS off_chain_vote_fetch_errors;
DROP TABLE IF EXISTS off_chain_vote_external_updates;
DROP TABLE IF EXISTS off_chain_vote_references;
DROP TABLE IF EXISTS off_chain_vote_authors;
DROP TABLE IF EXISTS off_chain_vote_d_rep_data;
DROP TABLE IF EXISTS off_chain_vote_gov_action_data;
DROP TABLE IF EXISTS off_chain_vote_data;
DROP TABLE IF EXISTS off_chain_pool_fetch_errors;
DROP TABLE IF EXISTS off_chain_pool_data;

-- Governance tables
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
DROP TABLE IF EXISTS voting_procedures;
DROP TABLE IF EXISTS gov_action_proposals;
DROP TABLE IF EXISTS param_proposals;
DROP TABLE IF EXISTS committee_hashes;
DROP TABLE IF EXISTS d_rep_hashes;
DROP TABLE IF EXISTS voting_anchors;

-- Staking and pool tables
DROP TABLE IF EXISTS epoch_params;
DROP TABLE IF EXISTS cost_models;
DROP TABLE IF EXISTS reserved_pool_tickers;
DROP TABLE IF EXISTS delisted_pools;
DROP TABLE IF EXISTS epoch_stake_progresses;
DROP TABLE IF EXISTS epoch_stakes;
DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS reward_rests;
DROP TABLE IF EXISTS rewards;
DROP TABLE IF EXISTS delegations;
DROP TABLE IF EXISTS stake_deregistrations;
DROP TABLE IF EXISTS stake_registrations;
DROP TABLE IF EXISTS pool_stats;
DROP TABLE IF EXISTS pool_owners;
DROP TABLE IF EXISTS pool_retires;
DROP TABLE IF EXISTS pool_relays;
DROP TABLE IF EXISTS pool_updates;
DROP TABLE IF EXISTS pool_metadata_refs;

-- Transaction related tables
DROP TABLE IF EXISTS tx_cbors;
DROP TABLE IF EXISTS extra_key_witnesses;
DROP TABLE IF EXISTS tx_metadata;
DROP TABLE IF EXISTS collateral_tx_outs;
DROP TABLE IF EXISTS reference_tx_ins;
DROP TABLE IF EXISTS collateral_tx_ins;
DROP TABLE IF EXISTS ma_tx_mints;
DROP TABLE IF EXISTS ma_tx_outs;
DROP TABLE IF EXISTS multi_assets;
DROP TABLE IF EXISTS redeemers;
DROP TABLE IF EXISTS tx_ins;
DROP TABLE IF EXISTS tx_outs;
DROP TABLE IF EXISTS redeemer_data;
DROP TABLE IF EXISTS data;  -- This is the 'datum' table
DROP TABLE IF EXISTS scripts;

-- Core tables
DROP TABLE IF EXISTS pool_hashes;
DROP TABLE IF EXISTS stake_addresses;
DROP TABLE IF EXISTS txes;
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS epoches;
DROP TABLE IF EXISTS slot_leaders;

-- Utility tables
DROP TABLE IF EXISTS ada_pots;
DROP TABLE IF EXISTS event_infos;
DROP TABLE IF EXISTS utxo_states;
DROP TABLE IF EXISTS utxo_deltas;
DROP TABLE IF EXISTS sync_checkpoints;
DROP TABLE IF EXISTS checkpoint_resume_logs;

SET FOREIGN_KEY_CHECKS = 1;