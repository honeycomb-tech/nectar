-- Add CASCADE DELETE rules for proper rollback support
-- This ensures data integrity when blocks are rolled back

-- ============================================
-- TRANSACTION CASCADE DELETES
-- ============================================

-- Ensure transactions are deleted when blocks are deleted
ALTER TABLE tx 
DROP FOREIGN KEY tx_ibfk_1,
ADD CONSTRAINT tx_block_id_fk 
FOREIGN KEY (block_id) REFERENCES block(id) ON DELETE CASCADE;

-- Ensure transaction outputs are deleted when transactions are deleted
ALTER TABLE tx_out
DROP FOREIGN KEY IF EXISTS tx_out_ibfk_1,
ADD CONSTRAINT tx_out_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Ensure transaction inputs are deleted when transactions are deleted
ALTER TABLE tx_in
DROP FOREIGN KEY IF EXISTS tx_in_ibfk_1,
ADD CONSTRAINT tx_in_tx_in_id_fk
FOREIGN KEY (tx_in_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- CERTIFICATE CASCADE DELETES
-- ============================================

-- Stake registrations
ALTER TABLE stake_registrations
DROP FOREIGN KEY IF EXISTS stake_registrations_ibfk_1,
ADD CONSTRAINT stake_registration_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Stake deregistrations
ALTER TABLE stake_deregistrations
DROP FOREIGN KEY IF EXISTS stake_deregistrations_ibfk_1,
ADD CONSTRAINT stake_deregistration_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Delegations
ALTER TABLE delegations
DROP FOREIGN KEY IF EXISTS delegations_ibfk_1,
ADD CONSTRAINT delegation_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Pool updates
ALTER TABLE pool_updates
DROP FOREIGN KEY IF EXISTS pool_updates_ibfk_1,
ADD CONSTRAINT pool_update_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Pool retirements
ALTER TABLE pool_retires
DROP FOREIGN KEY IF EXISTS pool_retires_ibfk_1,
ADD CONSTRAINT pool_retire_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- METADATA CASCADE DELETES
-- ============================================

-- Transaction metadata
ALTER TABLE tx_metadata
DROP FOREIGN KEY IF EXISTS tx_metadata_ibfk_1,
ADD CONSTRAINT tx_metadata_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- SCRIPT CASCADE DELETES
-- ============================================

-- Redeemers
ALTER TABLE redeemers
DROP FOREIGN KEY IF EXISTS redeemers_ibfk_1,
ADD CONSTRAINT redeemer_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- MULTI-ASSET CASCADE DELETES
-- ============================================

-- Multi-asset outputs
ALTER TABLE ma_tx_out
DROP FOREIGN KEY IF EXISTS ma_tx_out_ibfk_1,
ADD CONSTRAINT ma_tx_out_tx_out_id_fk
FOREIGN KEY (tx_out_id) REFERENCES tx_out(id) ON DELETE CASCADE;

-- Multi-asset mints
ALTER TABLE ma_tx_mint
DROP FOREIGN KEY IF EXISTS ma_tx_mint_ibfk_1,
ADD CONSTRAINT ma_tx_mint_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- GOVERNANCE CASCADE DELETES (Conway)
-- ============================================

-- Governance action proposals
ALTER TABLE gov_action_proposals
DROP FOREIGN KEY IF EXISTS gov_action_proposals_ibfk_1,
ADD CONSTRAINT gov_action_proposal_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Voting procedures
ALTER TABLE voting_procedures
DROP FOREIGN KEY IF EXISTS voting_procedures_ibfk_1,
ADD CONSTRAINT voting_procedure_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Treasury withdrawals
ALTER TABLE treasury_withdrawals
DROP FOREIGN KEY IF EXISTS treasury_withdrawals_ibfk_1,
ADD CONSTRAINT treasury_withdrawal_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- WITNESS CASCADE DELETES
-- ============================================

-- Extra key witnesses
ALTER TABLE extra_key_witnesses
DROP FOREIGN KEY IF EXISTS extra_key_witnesses_ibfk_1,
ADD CONSTRAINT extra_key_witness_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- WITHDRAWAL CASCADE DELETES
-- ============================================

-- Withdrawals
ALTER TABLE withdrawals
DROP FOREIGN KEY IF EXISTS withdrawals_ibfk_1,
ADD CONSTRAINT withdrawal_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- COLLATERAL CASCADE DELETES
-- ============================================

-- Collateral inputs
ALTER TABLE collateral_tx_in
DROP FOREIGN KEY IF EXISTS collateral_tx_in_ibfk_1,
ADD CONSTRAINT collateral_tx_in_tx_in_id_fk
FOREIGN KEY (tx_in_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Collateral outputs
ALTER TABLE collateral_tx_out
DROP FOREIGN KEY IF EXISTS collateral_tx_out_ibfk_1,
ADD CONSTRAINT collateral_tx_out_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Reference inputs
ALTER TABLE reference_tx_in
DROP FOREIGN KEY IF EXISTS reference_tx_in_ibfk_1,
ADD CONSTRAINT reference_tx_in_tx_in_id_fk
FOREIGN KEY (tx_in_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- EPOCH DATA CASCADE DELETES
-- ============================================

-- Ada pots should be deleted when blocks are deleted
ALTER TABLE ada_pots
DROP FOREIGN KEY IF EXISTS ada_pots_ibfk_1,
ADD CONSTRAINT ada_pots_block_id_fk
FOREIGN KEY (block_id) REFERENCES block(id) ON DELETE CASCADE;

-- Epoch parameters should be deleted when blocks are deleted
ALTER TABLE epoch_param
DROP FOREIGN KEY IF EXISTS epoch_param_ibfk_1,
ADD CONSTRAINT epoch_param_block_id_fk
FOREIGN KEY (block_id) REFERENCES block(id) ON DELETE CASCADE;

-- ============================================
-- INSTANT REWARDS CASCADE DELETES
-- ============================================

-- Instant rewards (MIR)
ALTER TABLE instant_rewards
DROP FOREIGN KEY IF EXISTS instant_rewards_ibfk_1,
ADD CONSTRAINT instant_reward_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- Pot transfers
ALTER TABLE pot_transfers
DROP FOREIGN KEY IF EXISTS pot_transfers_ibfk_1,
ADD CONSTRAINT pot_transfer_tx_id_fk
FOREIGN KEY (tx_id) REFERENCES tx(id) ON DELETE CASCADE;

-- ============================================
-- PARAMETER PROPOSAL CASCADE DELETES
-- ============================================

-- Parameter proposals
ALTER TABLE param_proposals
DROP FOREIGN KEY IF EXISTS param_proposals_ibfk_1,
ADD CONSTRAINT param_proposal_tx_id_fk
FOREIGN KEY (registered_tx_id) REFERENCES tx(id) ON DELETE CASCADE;