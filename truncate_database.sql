-- Nectar Database Truncation Script
-- This will clear all data but preserve schema

SET FOREIGN_KEY_CHECKS = 0;

-- Core tables
TRUNCATE TABLE blocks;
TRUNCATE TABLE txes;
TRUNCATE TABLE tx_outs;
TRUNCATE TABLE tx_ins;
TRUNCATE TABLE collateral_tx_ins;
TRUNCATE TABLE reference_tx_ins;
TRUNCATE TABLE collateral_tx_outs;
TRUNCATE TABLE tx_cbor;
TRUNCATE TABLE tx_metadata;
TRUNCATE TABLE extra_key_witnesses;
TRUNCATE TABLE checkpoints;

-- Asset tables
TRUNCATE TABLE multi_assets;
TRUNCATE TABLE ma_tx_mints;
TRUNCATE TABLE ma_tx_outs;

-- Script/Datum tables
TRUNCATE TABLE scripts;
TRUNCATE TABLE datum;
TRUNCATE TABLE redeemer_data;
TRUNCATE TABLE redeemers;

-- Staking tables
TRUNCATE TABLE stake_addresses;
TRUNCATE TABLE pool_hashes;
TRUNCATE TABLE pool_metadata_refs;
TRUNCATE TABLE pool_updates;
TRUNCATE TABLE pool_owners;
TRUNCATE TABLE pool_retires;
TRUNCATE TABLE pool_relays;
TRUNCATE TABLE stake_registrations;
TRUNCATE TABLE stake_deregistrations;
TRUNCATE TABLE delegations;
TRUNCATE TABLE rewards;
TRUNCATE TABLE reward_rests;
TRUNCATE TABLE withdrawals;
TRUNCATE TABLE pool_stats;
TRUNCATE TABLE delisted_pools;
TRUNCATE TABLE reserved_pool_tickers;

-- Epoch tables
TRUNCATE TABLE epochs;
TRUNCATE TABLE epoch_params;
TRUNCATE TABLE epoch_stakes;
TRUNCATE TABLE epoch_stake_progress;
TRUNCATE TABLE slot_leaders;
TRUNCATE TABLE ada_pots;
TRUNCATE TABLE treasuries;
TRUNCATE TABLE reserves;
TRUNCATE TABLE pot_transfers;
TRUNCATE TABLE epoch_states;

-- Governance tables
TRUNCATE TABLE drep_hashes;
TRUNCATE TABLE committee_hashes;
TRUNCATE TABLE committees;
TRUNCATE TABLE committee_members;
TRUNCATE TABLE committee_registrations;
TRUNCATE TABLE committee_de_registrations;
TRUNCATE TABLE constitutions;
TRUNCATE TABLE voting_anchors;
TRUNCATE TABLE voting_procedures;
TRUNCATE TABLE delegation_votes;
TRUNCATE TABLE gov_action_proposals;
TRUNCATE TABLE treasury_withdrawals;
TRUNCATE TABLE param_proposals;
TRUNCATE TABLE drep_distrs;

-- Off-chain data tables
TRUNCATE TABLE off_chain_pool_data;
TRUNCATE TABLE off_chain_pool_fetch_errors;
TRUNCATE TABLE off_chain_vote_data;
TRUNCATE TABLE off_chain_vote_gov_action_data;
TRUNCATE TABLE off_chain_vote_drep_data;
TRUNCATE TABLE off_chain_vote_authors;
TRUNCATE TABLE off_chain_vote_references;
TRUNCATE TABLE off_chain_vote_external_updates;
TRUNCATE TABLE off_chain_vote_fetch_errors;

-- Event info
TRUNCATE TABLE event_infos;

-- Cost models
TRUNCATE TABLE cost_models;

SET FOREIGN_KEY_CHECKS = 1;

-- Reset auto-increment counters
ALTER TABLE blocks AUTO_INCREMENT = 1;
ALTER TABLE txes AUTO_INCREMENT = 1;
ALTER TABLE tx_outs AUTO_INCREMENT = 1;
ALTER TABLE stake_addresses AUTO_INCREMENT = 1;
ALTER TABLE pool_hashes AUTO_INCREMENT = 1;

SELECT 'Database truncated successfully!' as Result;