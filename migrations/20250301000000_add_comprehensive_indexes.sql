-- Comprehensive performance indexes for Nectar
-- These indexes optimize query performance for common access patterns

-- ============================================
-- CORE TABLES INDEXES
-- ============================================

-- Block indexes
CREATE INDEX IF NOT EXISTS idx_block_epoch_no ON block(epoch_no);
CREATE INDEX IF NOT EXISTS idx_block_slot_no ON block(slot_no);
CREATE INDEX IF NOT EXISTS idx_block_time ON block(time);
CREATE INDEX IF NOT EXISTS idx_block_previous_id ON block(previous_id);

-- Transaction indexes
CREATE INDEX IF NOT EXISTS idx_tx_block_id_block_index ON tx(block_id, block_index);
CREATE INDEX IF NOT EXISTS idx_tx_valid_contract ON tx(valid_contract);
CREATE INDEX IF NOT EXISTS idx_tx_size ON tx(size);

-- Transaction output indexes
CREATE INDEX IF NOT EXISTS idx_tx_out_tx_id_index ON tx_out(tx_id, `index`);
CREATE INDEX IF NOT EXISTS idx_tx_out_stake_address_id ON tx_out(stake_address_id);
CREATE INDEX IF NOT EXISTS idx_tx_out_address ON tx_out(address);
CREATE INDEX IF NOT EXISTS idx_tx_out_payment_cred ON tx_out(payment_cred);

-- Transaction input indexes
CREATE INDEX IF NOT EXISTS idx_tx_in_tx_out ON tx_in(tx_out_id, tx_out_index);
CREATE INDEX IF NOT EXISTS idx_tx_in_tx_in_id ON tx_in(tx_in_id);
CREATE INDEX IF NOT EXISTS idx_tx_in_redeemer_id ON tx_in(redeemer_id);

-- ============================================
-- STAKING INDEXES
-- ============================================

-- Stake address indexes
CREATE INDEX IF NOT EXISTS idx_stake_address_hash_raw ON stake_addresses(hash_raw);
CREATE INDEX IF NOT EXISTS idx_stake_address_view ON stake_addresses(view);

-- Pool hash indexes
CREATE INDEX IF NOT EXISTS idx_pool_hash_hash_raw ON pool_hashes(hash_raw);
CREATE INDEX IF NOT EXISTS idx_pool_hash_view ON pool_hashes(view);

-- Delegation indexes
CREATE INDEX IF NOT EXISTS idx_delegation_addr_id ON delegations(addr_id);
CREATE INDEX IF NOT EXISTS idx_delegation_pool_hash_id ON delegations(pool_hash_id);
CREATE INDEX IF NOT EXISTS idx_delegation_active_epoch ON delegations(active_epoch_no);
CREATE INDEX IF NOT EXISTS idx_delegation_tx_id ON delegations(tx_id);

-- Reward indexes
CREATE INDEX IF NOT EXISTS idx_reward_addr_id_epoch ON rewards(addr_id, earned_epoch);
CREATE INDEX IF NOT EXISTS idx_reward_pool_id ON rewards(pool_id);
CREATE INDEX IF NOT EXISTS idx_reward_spendable_epoch ON rewards(spendable_epoch);

-- Pool update indexes
CREATE INDEX IF NOT EXISTS idx_pool_update_hash_id ON pool_updates(hash_id);
CREATE INDEX IF NOT EXISTS idx_pool_update_active_epoch ON pool_updates(active_epoch_no);

-- ============================================
-- MULTI-ASSET INDEXES
-- ============================================

-- Multi-asset output indexes
CREATE INDEX IF NOT EXISTS idx_ma_tx_out_tx_out_id ON ma_tx_out(tx_out_id);
CREATE INDEX IF NOT EXISTS idx_ma_tx_out_ident ON ma_tx_out(ident);

-- Multi-asset mint indexes
CREATE INDEX IF NOT EXISTS idx_ma_tx_mint_tx_id ON ma_tx_mint(tx_id);
CREATE INDEX IF NOT EXISTS idx_ma_tx_mint_ident ON ma_tx_mint(ident);
CREATE INDEX IF NOT EXISTS idx_ma_tx_mint_quantity ON ma_tx_mint(quantity);

-- ============================================
-- SCRIPT AND METADATA INDEXES
-- ============================================

-- Script indexes
CREATE INDEX IF NOT EXISTS idx_script_hash ON scripts(hash);
CREATE INDEX IF NOT EXISTS idx_script_type ON scripts(type);

-- Datum indexes
CREATE INDEX IF NOT EXISTS idx_datum_hash ON datum(hash);

-- Metadata indexes
CREATE INDEX IF NOT EXISTS idx_tx_metadata_tx_id ON tx_metadata(tx_id);
CREATE INDEX IF NOT EXISTS idx_tx_metadata_key ON tx_metadata(key);

-- ============================================
-- GOVERNANCE INDEXES (Conway)
-- ============================================

-- Governance action indexes
CREATE INDEX IF NOT EXISTS idx_gov_action_tx_id ON gov_action_proposals(tx_id);
CREATE INDEX IF NOT EXISTS idx_gov_action_type ON gov_action_proposals(type);
CREATE INDEX IF NOT EXISTS idx_gov_action_epoch ON gov_action_proposals(epoch_no);

-- Voting procedure indexes
CREATE INDEX IF NOT EXISTS idx_voting_procedure_tx_id ON voting_procedures(tx_id);
CREATE INDEX IF NOT EXISTS idx_voting_procedure_voter_role ON voting_procedures(voter_role);
CREATE INDEX IF NOT EXISTS idx_voting_procedure_gov_action_id ON voting_procedures(gov_action_proposal_id);

-- Committee indexes
CREATE INDEX IF NOT EXISTS idx_committee_epoch ON committee(epoch_no);
CREATE INDEX IF NOT EXISTS idx_committee_member_committee_id ON committee_members(committee_id);

-- ============================================
-- EPOCH DATA INDEXES
-- ============================================

-- Epoch indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_epoch_no ON epoch(no);
CREATE INDEX IF NOT EXISTS idx_epoch_start_time ON epoch(start_time);

-- Ada pots indexes
CREATE INDEX IF NOT EXISTS idx_ada_pots_block_id ON ada_pots(block_id);
CREATE INDEX IF NOT EXISTS idx_ada_pots_epoch_no ON ada_pots(epoch_no);

-- Epoch param indexes
CREATE INDEX IF NOT EXISTS idx_epoch_param_epoch_no ON epoch_param(epoch_no);
CREATE INDEX IF NOT EXISTS idx_epoch_param_block_id ON epoch_param(block_id);

-- ============================================
-- CERTIFICATE INDEXES
-- ============================================

-- Stake registration/deregistration indexes
CREATE INDEX IF NOT EXISTS idx_stake_registration_addr_id ON stake_registrations(addr_id);
CREATE INDEX IF NOT EXISTS idx_stake_registration_tx_id ON stake_registrations(tx_id);
CREATE INDEX IF NOT EXISTS idx_stake_deregistration_addr_id ON stake_deregistrations(addr_id);
CREATE INDEX IF NOT EXISTS idx_stake_deregistration_tx_id ON stake_deregistrations(tx_id);

-- ============================================
-- UTILITY INDEXES
-- ============================================

-- Withdrawal indexes
CREATE INDEX IF NOT EXISTS idx_withdrawal_addr_id ON withdrawals(addr_id);
CREATE INDEX IF NOT EXISTS idx_withdrawal_tx_id ON withdrawals(tx_id);

-- Collateral indexes
CREATE INDEX IF NOT EXISTS idx_collateral_tx_in_tx_in_id ON collateral_tx_in(tx_in_id);
CREATE INDEX IF NOT EXISTS idx_collateral_tx_out_tx_id ON collateral_tx_out(tx_id);

-- Reference input indexes
CREATE INDEX IF NOT EXISTS idx_reference_tx_in_tx_in_id ON reference_tx_in(tx_in_id);

-- Extra key witness indexes
CREATE INDEX IF NOT EXISTS idx_extra_key_witness_tx_id ON extra_key_witnesses(tx_id);

-- ============================================
-- COMPOSITE INDEXES FOR COMMON QUERIES
-- ============================================

-- Block retrieval by epoch and slot
CREATE INDEX IF NOT EXISTS idx_block_epoch_slot ON block(epoch_no, slot_no);

-- Transaction retrieval by block
CREATE INDEX IF NOT EXISTS idx_tx_block_composite ON tx(block_id, block_index, id);

-- Stake address delegation history
CREATE INDEX IF NOT EXISTS idx_delegation_composite ON delegations(addr_id, active_epoch_no, tx_id);

-- Pool performance queries
CREATE INDEX IF NOT EXISTS idx_pool_update_composite ON pool_updates(hash_id, active_epoch_no);

-- Multi-asset holdings by address
CREATE INDEX IF NOT EXISTS idx_ma_tx_out_composite ON ma_tx_out(tx_out_id, ident, quantity);

-- ============================================
-- CLEANUP DUPLICATE INDEXES
-- ============================================

-- Drop any duplicate indexes that might exist
DROP INDEX IF EXISTS idx_txes_block_id;
DROP INDEX IF EXISTS idx_tx_outs_tx_id;
DROP INDEX IF EXISTS idx_tx_ins_tx_in_id;
DROP INDEX IF EXISTS idx_stake_addresses_hash_raw;