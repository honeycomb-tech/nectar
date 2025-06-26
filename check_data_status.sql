-- Comprehensive Nectar Data Status Check

-- Basic blockchain data
SELECT 'BLOCKCHAIN DATA' as category, '' as detail, '' as status;
SELECT '', 'Blocks', CONCAT(FORMAT(COUNT(*), 0), ' (Latest: Epoch ', MAX(epoch_no), ', Slot ', MAX(slot_no), ')') FROM blocks;
SELECT '', 'Transactions', FORMAT(COUNT(*), 0) FROM txes;
SELECT '', 'TX Outputs', FORMAT(COUNT(*), 0) FROM tx_outs;
SELECT '', 'TX Inputs', FORMAT(COUNT(*), 0) FROM tx_ins;

-- Shelley staking data (Certificate-based - should work)
SELECT 'STAKING DATA (Certificates)', '', '';
SELECT '', 'Stake Addresses', FORMAT(COUNT(*), 0) FROM stake_addresses;
SELECT '', 'Stake Registrations', FORMAT(COUNT(*), 0) FROM stake_registrations;
SELECT '', 'Delegations', FORMAT(COUNT(*), 0) FROM delegations;
SELECT '', 'Active Delegations', CONCAT(FORMAT(COUNT(DISTINCT addr_hash), 0), ' addresses → ', FORMAT(COUNT(DISTINCT pool_hash), 0), ' pools') FROM delegations WHERE active_epoch_no <= (SELECT MAX(epoch_no) FROM blocks);

-- Pool data (Certificate-based - should work)
SELECT 'POOL DATA (Certificates)', '', '';
SELECT '', 'Pool Hashes', FORMAT(COUNT(*), 0) FROM pool_hashes;
SELECT '', 'Pool Updates', FORMAT(COUNT(*), 0) FROM pool_updates;
SELECT '', 'Pool Owners', FORMAT(COUNT(*), 0) FROM pool_owners;
SELECT '', 'Pool Retirements', FORMAT(COUNT(*), 0) FROM pool_retires;
SELECT '', 'Pool Metadata Refs', FORMAT(COUNT(*), 0) FROM pool_metadata_refs;

-- StateQuery data (Our implementation - should work after epoch 211)
SELECT 'STATEQUERY DATA', '', '';
SELECT '', 'Rewards', IF(COUNT(*) = 0, '❌ MISSING (Should have data after epoch 211)', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM rewards;
SELECT '', 'Pool Stats', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM pool_stats;
SELECT '', 'Epoch Stakes', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM epoch_stakes;
SELECT '', 'Treasury', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM treasury;
SELECT '', 'Reserves', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM reserves;
SELECT '', 'ADA Pots', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM ada_pots;

-- Withdrawals (Should work - comes from certificates)
SELECT 'WITHDRAWALS', '', '';
SELECT '', 'Withdrawals', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM withdrawals;

-- Smart contract data (Should work in Shelley+)
SELECT 'SMART CONTRACT DATA', '', '';
SELECT '', 'Scripts', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM scripts;
SELECT '', 'Datums', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM datums;
SELECT '', 'Redeemers', IF(COUNT(*) = 0, '❌ MISSING', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM redeemers;

-- Multi-asset data (Should work after Mary era)
SELECT 'MULTI-ASSET DATA', '', '';
SELECT '', 'Multi Assets', IF(COUNT(*) = 0, '❌ MISSING (Mary era)', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM multi_assets;
SELECT '', 'MA TX Outputs', IF(COUNT(*) = 0, '❌ MISSING (Mary era)', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM ma_tx_outs;
SELECT '', 'MA TX Mints', IF(COUNT(*) = 0, '❌ MISSING (Mary era)', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM ma_tx_mints;

-- Metadata (Disabled in config)
SELECT 'METADATA', '', '';
SELECT '', 'TX Metadata', IF(COUNT(*) = 0, '⚠️ Disabled in config', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM tx_metadata;
SELECT '', 'Off-chain Pool Data', IF(COUNT(*) = 0, '⚠️ Metadata disabled', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM off_chain_pool_data;

-- Governance (Conway era)
SELECT 'GOVERNANCE DATA', '', '';
SELECT '', 'Param Proposals', IF(COUNT(*) = 0, 'Not in Conway era yet', CONCAT('✅ ', FORMAT(COUNT(*), 0))) FROM param_proposals;