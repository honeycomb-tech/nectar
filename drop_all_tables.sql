-- Drop all Nectar tables for a fresh start
-- This will remove ALL data!

SET FOREIGN_KEY_CHECKS = 0;

-- Drop all tables in reverse dependency order
DROP TABLE IF EXISTS ma_tx_outs;
DROP TABLE IF EXISTS ma_tx_mints;
DROP TABLE IF EXISTS multi_assets;
DROP TABLE IF EXISTS redeemers;
DROP TABLE IF EXISTS scripts;
DROP TABLE IF EXISTS tx_metadata;
DROP TABLE IF EXISTS pool_stats;
DROP TABLE IF EXISTS rewards;
DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS stake_deregistrations;
DROP TABLE IF EXISTS stake_registrations;
DROP TABLE IF EXISTS pool_retires;
DROP TABLE IF EXISTS pool_owners;
DROP TABLE IF EXISTS pool_updates;
DROP TABLE IF EXISTS delegations;
DROP TABLE IF EXISTS pool_hashes;
DROP TABLE IF EXISTS stake_addresses;
DROP TABLE IF EXISTS tx_ins;
DROP TABLE IF EXISTS tx_outs;
DROP TABLE IF EXISTS txes;
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS slot_leaders;
DROP TABLE IF EXISTS epochs;
DROP TABLE IF EXISTS ada_pots;
DROP TABLE IF EXISTS epoch_params;
DROP TABLE IF EXISTS cost_models;
DROP TABLE IF EXISTS param_proposals;
DROP TABLE IF EXISTS pool_metadata_refs;
DROP TABLE IF EXISTS pool_relays;
DROP TABLE IF EXISTS reserved_pool_tickers;
DROP TABLE IF EXISTS collateral_tx_ins;
DROP TABLE IF EXISTS collateral_tx_outs;
DROP TABLE IF EXISTS reference_tx_ins;
DROP TABLE IF EXISTS extra_key_witnesses;
DROP TABLE IF EXISTS inline_datums;
DROP TABLE IF EXISTS reference_scripts;
DROP TABLE IF EXISTS committee_registrations;
DROP TABLE IF EXISTS committee_deregistrations;
DROP TABLE IF EXISTS drep_registrations;
DROP TABLE IF EXISTS drep_deregistrations;
DROP TABLE IF EXISTS voting_anchors;
DROP TABLE IF EXISTS gov_action_proposals;
DROP TABLE IF EXISTS treasury_withdrawals;
DROP TABLE IF EXISTS committee_members;
DROP TABLE IF EXISTS voting_procedures;
DROP TABLE IF EXISTS drep_hashes;
DROP TABLE IF EXISTS delegation_votes;
DROP TABLE IF EXISTS committee_hashes;
DROP TABLE IF EXISTS constitution;
DROP TABLE IF EXISTS certificates;
DROP TABLE IF EXISTS datums;
DROP TABLE IF EXISTS metadata_keys;
DROP TABLE IF EXISTS metadata_labels;
DROP TABLE IF EXISTS metadata_schemas;
DROP TABLE IF EXISTS metadata_values;
DROP TABLE IF EXISTS plutus_data;
DROP TABLE IF EXISTS script_hashes;
DROP TABLE IF EXISTS tx_cbor;
DROP TABLE IF EXISTS tx_witnesses;
DROP TABLE IF EXISTS utxo_view;
DROP TABLE IF EXISTS block_tips;
DROP TABLE IF EXISTS chain_tips;
DROP TABLE IF EXISTS rollback_history;
DROP TABLE IF EXISTS sync_status;
DROP TABLE IF EXISTS error_logs;
DROP TABLE IF EXISTS performance_metrics;

SET FOREIGN_KEY_CHECKS = 1;

SELECT 'All tables dropped successfully!' as status;