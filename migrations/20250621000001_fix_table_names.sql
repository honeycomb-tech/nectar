-- Fix table name mismatches between migrations and models
-- This migration renames all tables to match the GORM model expectations

-- Core tables
ALTER TABLE block RENAME TO blocks;
ALTER TABLE tx RENAME TO txes;
ALTER TABLE tx_in RENAME TO tx_ins;
ALTER TABLE tx_out RENAME TO tx_outs;
ALTER TABLE tx_cbor RENAME TO tx_cbors;

-- Collateral tables
ALTER TABLE collateral_tx_in RENAME TO collateral_tx_ins;
ALTER TABLE collateral_tx_out RENAME TO collateral_tx_outs;

-- Stake/Pool tables
ALTER TABLE stake_address RENAME TO stake_addresses;
ALTER TABLE stake_registration RENAME TO stake_registrations;
ALTER TABLE stake_deregistration RENAME TO stake_deregistrations;
ALTER TABLE pool_hash RENAME TO pool_hashes;
ALTER TABLE pool_update RENAME TO pool_updates;
ALTER TABLE pool_owner RENAME TO pool_owners;
ALTER TABLE pool_relay RENAME TO pool_relays;
ALTER TABLE pool_retire RENAME TO pool_retires;
ALTER TABLE pool_stat RENAME TO pool_stats;
ALTER TABLE pool_metadata_ref RENAME TO pool_metadata_refs;
ALTER TABLE delisted_pool RENAME TO delisted_pools;
ALTER TABLE reserved_pool_ticker RENAME TO reserved_pool_tickers;

-- Delegation tables
ALTER TABLE delegation RENAME TO delegations;
ALTER TABLE delegation_vote RENAME TO delegation_votes;

-- Epoch tables
ALTER TABLE epoch RENAME TO epoches;
ALTER TABLE epoch_param RENAME TO epoch_params;
ALTER TABLE epoch_stake RENAME TO epoch_stakes;
ALTER TABLE epoch_state RENAME TO epoch_states;

-- Reward tables
ALTER TABLE reward RENAME TO rewards;
ALTER TABLE reward_rest RENAME TO reward_rests;
ALTER TABLE withdrawal RENAME TO withdrawals;

-- Script/Datum tables
ALTER TABLE script RENAME TO scripts;
ALTER TABLE datum RENAME TO data;
ALTER TABLE redeemer RENAME TO redeemers;

-- Multi-asset tables
ALTER TABLE multi_asset RENAME TO multi_assets;
ALTER TABLE ma_tx_mint RENAME TO ma_tx_mints;
ALTER TABLE ma_tx_out RENAME TO ma_tx_outs;

-- Governance tables (Conway era)
ALTER TABLE committee RENAME TO committees;
ALTER TABLE committee_hash RENAME TO committee_hashes;
ALTER TABLE committee_member RENAME TO committee_members;
ALTER TABLE committee_registration RENAME TO committee_registrations;
ALTER TABLE committee_de_registration RENAME TO committee_deregistrations;
ALTER TABLE constitution RENAME TO constitutions;
ALTER TABLE drep_hash RENAME TO d_rep_hashes;
ALTER TABLE drep_distr RENAME TO d_rep_distrs;
ALTER TABLE gov_action_proposal RENAME TO gov_action_proposals;
ALTER TABLE param_proposal RENAME TO param_proposals;
ALTER TABLE voting_procedure RENAME TO voting_procedures;
ALTER TABLE treasury_withdrawal RENAME TO treasury_withdrawals;

-- Off-chain data tables
ALTER TABLE off_chain_vote_author RENAME TO off_chain_vote_authors;
ALTER TABLE off_chain_vote_drep_data RENAME TO off_chain_vote_d_rep_data;
ALTER TABLE off_chain_vote_external_update RENAME TO off_chain_vote_external_updates;
ALTER TABLE off_chain_vote_reference RENAME TO off_chain_vote_references;

-- Other tables
ALTER TABLE cost_model RENAME TO cost_models;
ALTER TABLE event_info RENAME TO event_infos;
ALTER TABLE extra_key_witness RENAME TO extra_key_witnesses;
ALTER TABLE pot_transfer RENAME TO pot_transfers;
ALTER TABLE reference_tx_in RENAME TO reference_tx_ins;
ALTER TABLE reserve RENAME TO reserves;
ALTER TABLE slot_leader RENAME TO slot_leaders;
ALTER TABLE treasury RENAME TO treasuries;

-- Note: These tables already have correct names or are handled differently:
-- ada_pots (correct)
-- checkpoint_resume_log (correct)
-- sync_checkpoints (correct)
-- epoch_stake_progress -> epoch_stake_progresses (already correct in newer migration)
-- redeemer_data (correct)
-- required_signers (correct)
-- tx_metadata (correct)
-- voting_anchors (correct)
-- off_chain_pool_data (correct)
-- off_chain_pool_fetch_error (correct)
-- off_chain_vote_data (correct)
-- off_chain_vote_fetch_errors (correct)
-- off_chain_vote_gov_action_data (correct)