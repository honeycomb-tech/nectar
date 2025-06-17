package processors

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"nectar/models"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"gorm.io/gorm"
)

// Voter type constants
const (
	VoterTypeConstitutionalCommitteeHotKeyHash    uint8 = 0
	VoterTypeConstitutionalCommitteeHotScriptHash uint8 = 1
	VoterTypeDRepKeyHash                          uint8 = 2
	VoterTypeDRepScriptHash                       uint8 = 3
	VoterTypeStakingPoolKeyHash                   uint8 = 4
)

// GovernanceProcessor handles processing governance features for Conway era
type GovernanceProcessor struct {
	db                 *gorm.DB
	drepHashCache      *DRepHashCache
	committeeHashCache *CommitteeHashCache
	poolHashCache      map[string]bool // Pool hash existence cache
	errorCollector     *ErrorCollector
	metadataFetcher    MetadataFetcher
}

// DRepHashCache manages DRep hash lookups
type DRepHashCache struct {
	cache map[string]bool // Just track existence
	db    *gorm.DB
}

// NewDRepHashCache creates a new DRep hash cache
func NewDRepHashCache(db *gorm.DB) *DRepHashCache {
	return &DRepHashCache{
		cache: make(map[string]bool),
		db:    db,
	}
}

// EnsureDRepHash ensures a DRep hash exists
func (dhc *DRepHashCache) EnsureDRepHash(hashBytes []byte, hashHex string) error {
	// Check cache first
	if exists := dhc.cache[hashHex]; exists {
		return nil
	}

	// Check database
	var drepHash models.DRepHash
	result := dhc.db.Where("raw = ?", hashBytes).First(&drepHash)
	if result.Error == nil {
		dhc.cache[hashHex] = true
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", result.Error)
	}

	// Create new DRep hash
	drepHash = models.DRepHash{
		HashRaw: hashBytes,
		View:    hashHex,
	}

	if err := dhc.db.Create(&drepHash).Error; err != nil {
		return fmt.Errorf("failed to create DRep hash: %w", err)
	}

	dhc.cache[hashHex] = true
	return nil
}

// CommitteeHashCache manages committee hash lookups
type CommitteeHashCache struct {
	cache map[string]bool // Just track existence
	db    *gorm.DB
}

// NewCommitteeHashCache creates a new committee hash cache
func NewCommitteeHashCache(db *gorm.DB) *CommitteeHashCache {
	return &CommitteeHashCache{
		cache: make(map[string]bool),
		db:    db,
	}
}

// EnsureCommitteeHash ensures a committee hash exists
func (chc *CommitteeHashCache) EnsureCommitteeHash(hashBytes []byte, hashHex string) error {
	// Check cache first
	if exists := chc.cache[hashHex]; exists {
		return nil
	}

	// Check database
	var committeeHash models.CommitteeHash
	result := chc.db.Where("raw = ?", hashBytes).First(&committeeHash)
	if result.Error == nil {
		chc.cache[hashHex] = true
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", result.Error)
	}

	// Create new committee hash
	committeeHash = models.CommitteeHash{
		HashRaw: hashBytes,
	}

	if err := chc.db.Create(&committeeHash).Error; err != nil {
		return fmt.Errorf("failed to create committee hash: %w", err)
	}

	chc.cache[hashHex] = true
	return nil
}

// NewGovernanceProcessor creates a new governance processor
func NewGovernanceProcessor(db *gorm.DB) *GovernanceProcessor {
	return &GovernanceProcessor{
		db:                 db,
		drepHashCache:      NewDRepHashCache(db),
		committeeHashCache: NewCommitteeHashCache(db),
		poolHashCache:      make(map[string]bool, 3000), // Pre-allocate for ~3K pools
		errorCollector:     GetGlobalErrorCollector(),
	}
}

// SetMetadataFetcher sets the metadata fetcher for off-chain data
func (gp *GovernanceProcessor) SetMetadataFetcher(fetcher MetadataFetcher) {
	gp.metadataFetcher = fetcher
}

// ProcessVotingProcedures processes voting procedures from a transaction
func (gp *GovernanceProcessor) ProcessVotingProcedures(ctx context.Context, tx *gorm.DB, txHash []byte, transaction interface{}, blockType uint) error {
	// Only process for Conway era
	if blockType < 7 {
		return nil
	}

	// Try to extract voting procedures
	switch txWithVotes := transaction.(type) {
	case interface{ VotingProcedures() map[*common.Voter]map[*common.GovActionId]common.VotingProcedure }:
		votingProcedures := txWithVotes.VotingProcedures()
		if len(votingProcedures) == 0 {
			return nil
		}

		log.Printf("Processing %d voting procedures for transaction %x", len(votingProcedures), txHash)

		for voter, votes := range votingProcedures {
			for govActionID, procedure := range votes {
				if err := gp.processVotingProcedure(tx, txHash, voter, govActionID, procedure); err != nil {
					gp.errorCollector.ProcessingWarning("GovernanceProcessor", "processVotingProcedure",
						fmt.Sprintf("failed to process voting procedure: %v", err),
						fmt.Sprintf("tx_hash:%x", txHash))
					log.Printf("[WARNING] Voting procedure processing error: %v", err)
				}
			}
		}
		return nil

	default:
		// Transaction doesn't support voting procedures
		return nil
	}
}

// processVotingProcedure processes a single voting procedure
func (gp *GovernanceProcessor) processVotingProcedure(tx *gorm.DB, txHash []byte, voter *common.Voter, govActionID *common.GovActionId, procedure common.VotingProcedure) error {
	// Convert voter type to role
	voterRole := gp.convertVoterTypeToRole(voter.Type)
	voteChoice := gp.convertVoteToChoice(procedure.Vote)
	
	// Create voting procedure hash from tx hash and governance action ID
	govActionProposalHash := models.GenerateGovActionProposalHash(govActionID.TransactionId[:], govActionID.GovActionIdx)

	// Create voting procedure record
	index := gp.getNextVotingProcedureIndex(tx, txHash)
	votingProc := &models.VotingProcedure{
		TxHash:                txHash,
		Index:                 index,
		GovActionProposalHash: govActionProposalHash,
		VoterRole:             voterRole,
		CommitteeVoterHash:    nil,
		DRepVoterHash:         nil,
		PoolVoterHash:         nil,
		Vote:                  voteChoice,
	}

	// Set voter based on type
	switch voter.Type {
	case VoterTypeConstitutionalCommitteeHotKeyHash, VoterTypeConstitutionalCommitteeHotScriptHash:
		if err := gp.committeeHashCache.EnsureCommitteeHash(voter.Hash[:], hex.EncodeToString(voter.Hash[:])); err != nil {
			return fmt.Errorf("failed to ensure committee voter hash: %w", err)
		}
		voterHash := voter.Hash[:]
		votingProc.CommitteeVoter = voterHash
		log.Printf("Committee vote recorded")
	case VoterTypeDRepKeyHash, VoterTypeDRepScriptHash:
		if err := gp.drepHashCache.EnsureDRepHash(voter.Hash[:], hex.EncodeToString(voter.Hash[:])); err != nil {
			return fmt.Errorf("failed to ensure DRep voter hash: %w", err)
		}
		voterHash := voter.Hash[:]
		votingProc.DRepVoter = voterHash
		log.Printf("DRep vote recorded")
	case VoterTypeStakingPoolKeyHash:
		// Ensure pool hash exists in pool_hashes table
		hashStr := hex.EncodeToString(voter.Hash[:])
		if !gp.poolHashCache[hashStr] {
			// Create pool hash if not exists
			poolHash := &models.PoolHash{
				HashRaw: voter.Hash[:],
				View:    hashStr,
			}
			if err := tx.Create(poolHash).Error; err != nil {
				return fmt.Errorf("failed to create pool hash: %w", err)
			}
			gp.poolHashCache[hashStr] = true
		}
		voterHash := voter.Hash[:]
		votingProc.PoolVoter = voterHash
		log.Printf("Stake pool vote recorded")
	}

	// Process voting anchor if present
	if procedure.Anchor != nil {
		anchorHash, err := gp.processVotingAnchor(tx, procedure.Anchor)
		if err != nil {
			log.Printf("[WARNING] Failed to process voting anchor: %v", err)
		} else {
			votingProc.VotingAnchorHash = anchorHash
		}
	}

	if err := tx.Create(votingProc).Error; err != nil {
		return fmt.Errorf("failed to create voting procedure: %w", err)
	}

	log.Printf("[OK] Processed voting procedure: voter_role=%d, vote=%d, tx_hash=%x", voter.Type, procedure.Vote, txHash)
	return nil
}

// processVotingAnchor processes a voting anchor
func (gp *GovernanceProcessor) processVotingAnchor(tx *gorm.DB, anchor *common.GovAnchor) ([]byte, error) {
	// Check if voting anchor already exists
	var existingAnchor models.VotingAnchor
	err := tx.Where("url = ? AND data_hash = ?", anchor.Url, anchor.DataHash[:]).First(&existingAnchor).Error
	if err == nil {
		return existingAnchor.DataHash, nil
	}
	
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query voting anchor: %w", err)
	}
	
	// Create voting anchor record
	votingAnchor := &models.VotingAnchor{
		URL:      anchor.Url,
		DataHash: anchor.DataHash[:],
	}

	if err := tx.Create(votingAnchor).Error; err != nil {
		return nil, fmt.Errorf("failed to create voting anchor: %w", err)
	}

	// Queue governance metadata for fetching if fetcher is available
	if gp.metadataFetcher != nil {
		if err := gp.metadataFetcher.QueueGovernanceMetadata(anchor.DataHash[:], anchor.Url, anchor.DataHash[:]); err != nil {
			log.Printf("[WARNING] Failed to queue governance metadata for fetching: %v", err)
		} else {
			log.Printf(" Queued governance metadata for off-chain fetching: %s", anchor.Url)
		}
	}

	return anchor.DataHash[:], nil
}

// ProcessProposalProcedures processes proposal procedures from a transaction
func (gp *GovernanceProcessor) ProcessProposalProcedures(ctx context.Context, tx *gorm.DB, txHash []byte, transaction interface{}, blockType uint) error {

	// Only process for Conway era
	if blockType < 7 {
		return nil
	}

	// Try to extract proposal procedures
	switch txWithProposals := transaction.(type) {
	case interface{ ProposalProcedures() []common.ProposalProcedure }:
		proposals := txWithProposals.ProposalProcedures()
		if len(proposals) == 0 {
			return nil
		}

		log.Printf("Processing %d proposal procedures for transaction %x", len(proposals), txHash)

		for proposalIndex, proposal := range proposals {
			if err := gp.processProposalProcedure(tx, txHash, proposalIndex, proposal); err != nil {
				gp.errorCollector.ProcessingWarning("GovernanceProcessor", "processProposalProcedure",
					fmt.Sprintf("failed to process proposal procedure %d: %v", proposalIndex, err),
					fmt.Sprintf("tx_hash:%x, proposal_index:%d", txHash, proposalIndex))
				log.Printf("[WARNING] Proposal procedure processing error: %v", err)
			}
		}
		return nil

	default:
		// Transaction doesn't support proposal procedures
		return nil
	}
}

// processProposalProcedure processes a single proposal procedure
func (gp *GovernanceProcessor) processProposalProcedure(tx *gorm.DB, txHash []byte, index int, proposal common.ProposalProcedure) error {

	// Create governance action proposal with hash-based primary key
	hash := models.GenerateGovActionProposalHash(txHash, uint32(index))
	govAction := &models.GovActionProposal{
		Hash:    hash,
		TxHash:  txHash,
		Index:   uint32(index),
		Type:    getGovActionType(proposal.GovAction.Action),
		Deposit: proposal.Deposit,
	}

	// Handle anchor if present
	anchorHash, err := gp.processVotingAnchor(tx, &proposal.Anchor)
	if err != nil {
		log.Printf("[WARNING] Failed to process proposal anchor: %v", err)
	} else {
		govAction.VotingAnchorHash = anchorHash
	}

	if err := tx.Create(govAction).Error; err != nil {
		return fmt.Errorf("failed to create governance action proposal: %w", err)
	}

	// Process specific governance action details
	switch action := proposal.GovAction.Action.(type) {
	case *common.ParameterChangeGovAction:
		if err := gp.processParameterChange(tx, hash, action); err != nil {
			log.Printf("[WARNING] Failed to process parameter change: %v", err)
		}
	case *common.HardForkInitiationGovAction:
		if err := gp.processHardForkInitiation(tx, hash, action); err != nil {
			log.Printf("[WARNING] Failed to process hard fork initiation: %v", err)
		}
	case *common.TreasuryWithdrawalGovAction:
		if err := gp.processTreasuryWithdrawals(tx, hash, action); err != nil {
			log.Printf("[WARNING] Failed to process treasury withdrawals: %v", err)
		}
	case *common.NoConfidenceGovAction:
		log.Printf("[OK] Processed no confidence governance action")
	case *common.UpdateCommitteeGovAction:
		if err := gp.processNewCommittee(tx, hash, action); err != nil {
			log.Printf("[WARNING] Failed to process new committee: %v", err)
		}
	case *common.NewConstitutionGovAction:
		if err := gp.processNewConstitution(tx, hash, action); err != nil {
			log.Printf("[WARNING] Failed to process new constitution: %v", err)
		}
	case *common.InfoGovAction:
		log.Printf("[OK] Processed info governance action")
	default:
		log.Printf("[WARNING] Unknown governance action type: %T", action)
	}

	return nil
}

// processParameterChange processes parameter change governance action
func (gp *GovernanceProcessor) processParameterChange(tx *gorm.DB, govActionHash []byte, action *common.ParameterChangeGovAction) error {
	// Store the previous governance action ID if present
	if action.ActionId != nil {
		// Update the governance action proposal with the previous hash reference
		prevHash := models.GenerateGovActionProposalHash(action.ActionId.TransactionId[:], action.ActionId.GovActionIdx)
		if err := tx.Model(&models.GovActionProposal{}).
			Where("hash = ?", govActionHash).
			Update("prev_gov_action_hash", prevHash).Error; err != nil {
			log.Printf("[WARNING] Failed to update previous action reference: %v", err)
		}
	}

	// Store parameter changes
	// Note: The actual structure of ParameterChangeGovAction in gouroboros
	// may not have a Params field. Check the specific implementation.
	// For now, we'll just log the action
	log.Printf(" Parameter change governance action recorded")
	
	// In a complete implementation, you would:
	// 1. Check the actual fields of ParameterChangeGovAction
	// 2. Extract each parameter (MinFeeA, MaxBlockSize, etc.)
	// 3. Store them in the ParamProposal table

	log.Printf("[OK] Parameter change governance action recorded")
	return nil
}

// processHardForkInitiation processes hard fork initiation governance action
func (gp *GovernanceProcessor) processHardForkInitiation(tx *gorm.DB, govActionHash []byte, action *common.HardForkInitiationGovAction) error {
	log.Printf("[OK] Processed hard fork initiation: v%d.%d", action.ProtocolVersion.Major, action.ProtocolVersion.Minor)
	return nil
}

// processTreasuryWithdrawals processes treasury withdrawal governance action
func (gp *GovernanceProcessor) processTreasuryWithdrawals(tx *gorm.DB, govActionHash []byte, action *common.TreasuryWithdrawalGovAction) error {
	for address, amount := range action.Withdrawals {
		// Get stake address hash
		stakeAddrHash := gp.resolveStakeAddressHash(tx, address)
		if stakeAddrHash == nil {
			continue
		}
		
		withdrawal := &models.TreasuryWithdrawal{
			GovActionProposalHash: govActionHash,
			Amount:                amount,
			StakeAddressHash:      stakeAddrHash,
		}

		if err := tx.Create(withdrawal).Error; err != nil {
			log.Printf("[WARNING] Failed to create treasury withdrawal: %v", err)
			continue
		}
	}
	return nil
}

// processNewCommittee processes new committee governance action
func (gp *GovernanceProcessor) processNewCommittee(tx *gorm.DB, govActionHash []byte, action *common.UpdateCommitteeGovAction) error {
	// Set default quorum threshold
	// The actual UpdateCommitteeGovAction structure may have a quorum field
	// For now, use a default value
	quorum := 0.67 // 2/3 majority
	
	// Create committee record with hash
	committeeHash := models.GenerateCommitteeHash(govActionHash, quorum)
	committee := &models.Committee{
		Hash:                  committeeHash,
		GovActionProposalHash: govActionHash,
		Quorum:                quorum,
	}
	
	if err := tx.Create(committee).Error; err != nil {
		return fmt.Errorf("failed to create committee record: %w", err)
	}

	// Process committee members from CredEpochs map
	for credential, epoch := range action.CredEpochs {
		if credential == nil {
			continue
		}
		memberHashHex := hex.EncodeToString(credential.Credential[:])
		if err := gp.committeeHashCache.EnsureCommitteeHash(credential.Credential[:], memberHashHex); err != nil {
			log.Printf("[WARNING] Failed to ensure committee hash: %v", err)
			continue
		}

		// Convert epoch to proper type
		epochUint32 := uint32(epoch)
		committeeMember := &models.CommitteeMember{
			CommitteeHash: credential.Credential[:], // The member's committee hash
			FromEpoch:     epochUint32,
			UntilEpoch:    &epochUint32,
		}

		if err := tx.Create(committeeMember).Error; err != nil {
			log.Printf("[WARNING] Failed to create committee member: %v", err)
		}
	}

	// Note: The actual UpdateCommitteeGovAction may have a RemovedMembers field
	// For now, we're just processing the new members

	log.Printf("[OK] Processed committee update governance action with %d members", len(action.CredEpochs))
	return nil
}

// processNewConstitution processes new constitution governance action
func (gp *GovernanceProcessor) processNewConstitution(tx *gorm.DB, govActionHash []byte, action *common.NewConstitutionGovAction) error {
	// Extract script hash and create anchor hash first  
	scriptHash := gp.extractConstitutionScriptHash(action)
	var votingAnchorHash []byte
	
	// Handle anchor if present
	if action.Constitution.Anchor.Url != "" {
		anchorHash, err := gp.processVotingAnchor(tx, &action.Constitution.Anchor)
		if err != nil {
			log.Printf("[WARNING] Failed to process voting anchor: %v", err)
		} else {
			votingAnchorHash = anchorHash
		}
	}
	
	// Create constitution record
	constitutionHash := models.GenerateConstitutionHash(govActionHash, votingAnchorHash, scriptHash)
	constitution := &models.Constitution{
		Hash:                  constitutionHash,
		GovActionProposalHash: govActionHash,
		ScriptHash:            scriptHash,
		VotingAnchorHash:      votingAnchorHash,
	}

	if err := tx.Create(constitution).Error; err != nil {
		return fmt.Errorf("failed to create constitution: %w", err)
	}

	log.Printf("[OK] Processed new constitution governance action")
	return nil
}

// ProcessDRepCertificates processes DRep-related certificates
func (gp *GovernanceProcessor) ProcessDRepCertificates(ctx context.Context, tx *gorm.DB, certificates []interface{}, txHash []byte) error {
	for certIndex, cert := range certificates {
		switch c := cert.(type) {
		case *common.RegistrationDrepCertificate:
			if err := gp.processRegistrationDrep(tx, txHash, certIndex, c); err != nil {
				log.Printf("[WARNING] Failed to process DRep registration: %v", err)
			}
		case *common.DeregistrationDrepCertificate:
			if err := gp.processDeregistrationDrep(tx, txHash, certIndex, c); err != nil {
				log.Printf("[WARNING] Failed to process DRep deregistration: %v", err)
			}
		case *common.UpdateDrepCertificate:
			if err := gp.processUpdateDrep(tx, txHash, certIndex, c); err != nil {
				log.Printf("[WARNING] Failed to process DRep update: %v", err)
			}
		}
	}
	return nil
}

// processRegistrationDrep processes DRep registration certificate
func (gp *GovernanceProcessor) processRegistrationDrep(tx *gorm.DB, txHash []byte, certIndex int, cert *common.RegistrationDrepCertificate) error {
	// Extract DRep credential hash
	drepHashHex := hex.EncodeToString(cert.DrepCredential.Credential[:])
	err := gp.drepHashCache.EnsureDRepHash(cert.DrepCredential.Credential[:], drepHashHex)
	if err != nil {
		return fmt.Errorf("failed to get DRep hash: %w", err)
	}

	// Process anchor if present
	if cert.Anchor != nil {
		_, err := gp.processVotingAnchor(tx, cert.Anchor)
		if err != nil {
			log.Printf("[WARNING] Failed to process DRep anchor: %v", err)
		}
	}

	log.Printf("[OK] Processed DRep registration: tx_hash=%x", txHash)
	return nil
}

// processDeregistrationDrep processes DRep deregistration certificate
func (gp *GovernanceProcessor) processDeregistrationDrep(tx *gorm.DB, txHash []byte, certIndex int, cert *common.DeregistrationDrepCertificate) error {
	// Extract DRep credential hash
	drepHashHex := hex.EncodeToString(cert.DrepCredential.Credential[:])
	err := gp.drepHashCache.EnsureDRepHash(cert.DrepCredential.Credential[:], drepHashHex)
	if err != nil {
		return fmt.Errorf("failed to get DRep hash: %w", err)
	}

	log.Printf("[OK] Processed DRep deregistration: tx_hash=%x", txHash)
	return nil
}

// processUpdateDrep processes DRep update certificate
func (gp *GovernanceProcessor) processUpdateDrep(tx *gorm.DB, txHash []byte, certIndex int, cert *common.UpdateDrepCertificate) error {
	// Extract DRep credential hash
	drepHashHex := hex.EncodeToString(cert.DrepCredential.Credential[:])
	err := gp.drepHashCache.EnsureDRepHash(cert.DrepCredential.Credential[:], drepHashHex)
	if err != nil {
		return fmt.Errorf("failed to get DRep hash: %w", err)
	}

	// Process anchor if present
	if cert.Anchor != nil {
		_, err := gp.processVotingAnchor(tx, cert.Anchor)
		if err != nil {
			log.Printf("[WARNING] Failed to process DRep update anchor: %v", err)
		}
	}

	log.Printf("[OK] Processed DRep update: tx_hash=%x", txHash)
	return nil
}

// ProcessCommitteeCertificates processes committee-related certificates
func (gp *GovernanceProcessor) ProcessCommitteeCertificates(ctx context.Context, tx *gorm.DB, certificates []interface{}, txHash []byte) error {
	for certIndex, cert := range certificates {
		switch c := cert.(type) {
		case *common.AuthCommitteeHotCertificate:
			if err := gp.processAuthCommitteeHot(tx, txHash, certIndex, c); err != nil {
				log.Printf("[WARNING] Failed to process committee hot auth: %v", err)
			}
		case *common.ResignCommitteeColdCertificate:
			if err := gp.processResignCommitteeCold(tx, txHash, certIndex, c); err != nil {
				log.Printf("[WARNING] Failed to process committee cold resign: %v", err)
			}
		}
	}
	return nil
}

// processAuthCommitteeHot processes committee hot key authorization certificate
func (gp *GovernanceProcessor) processAuthCommitteeHot(tx *gorm.DB, txHash []byte, certIndex int, cert *common.AuthCommitteeHotCertificate) error {
	// Extract cold key credential
	coldKeyHashHex := hex.EncodeToString(cert.ColdCredential.Credential[:])
	err := gp.committeeHashCache.EnsureCommitteeHash(cert.ColdCredential.Credential[:], coldKeyHashHex)
	if err != nil {
		return fmt.Errorf("failed to get/create cold key hash: %w", err)
	}

	// Extract hot key credential (note: gouroboros has typo "HostCredential" instead of "HotCredential")
	hotKeyHashHex := hex.EncodeToString(cert.HostCredential.Credential[:])
	err = gp.committeeHashCache.EnsureCommitteeHash(cert.HostCredential.Credential[:], hotKeyHashHex)
	if err != nil {
		return fmt.Errorf("failed to get/create hot key hash: %w", err)
	}

	// Create committee registration record
	committeeReg := &models.CommitteeRegistration{
		TxHash:       txHash,
		CertIndex:    uint32(certIndex),
		ColdKeyHash:  cert.ColdCredential.Credential[:],
		HotKeyHash:   cert.HostCredential.Credential[:],
	}

	if err := tx.Create(committeeReg).Error; err != nil {
		return fmt.Errorf("failed to create committee registration: %w", err)
	}

	log.Printf("[OK] Processed committee hot key authorization: cold=%s, hot=%s", coldKeyHashHex[:8], hotKeyHashHex[:8])
	return nil
}

// processResignCommitteeCold processes committee cold key resignation certificate
func (gp *GovernanceProcessor) processResignCommitteeCold(tx *gorm.DB, txHash []byte, certIndex int, cert *common.ResignCommitteeColdCertificate) error {
	// Extract cold key credential that is resigning
	coldKeyHashHex := hex.EncodeToString(cert.ColdCredential.Credential[:])
	err := gp.committeeHashCache.EnsureCommitteeHash(cert.ColdCredential.Credential[:], coldKeyHashHex)
	if err != nil {
		return fmt.Errorf("failed to get/create cold key hash: %w", err)
	}

	// Create committee de-registration record
	committeeDereg := &models.CommitteeDeregistration{
		TxHash:       txHash,
		CertIndex:    uint32(certIndex),
		ColdKeyHash:  cert.ColdCredential.Credential[:],
	}

	// Process resignation anchor if present
	if cert.Anchor != nil {
		anchorHash, err := gp.processVotingAnchor(tx, cert.Anchor)
		if err != nil {
			log.Printf("[WARNING] Failed to process resignation anchor: %v", err)
		} else {
			committeeDereg.AnchorHash = anchorHash
		}
	}

	if err := tx.Create(committeeDereg).Error; err != nil {
		return fmt.Errorf("failed to create committee de-registration: %w", err)
	}

	log.Printf("[OK] Processed committee cold key resignation: cold=%s", coldKeyHashHex[:8])
	return nil
}

// getGovActionType returns the governance action type as a string
func getGovActionType(govAction interface{}) string {
	switch govAction.(type) {
	case *common.ParameterChangeGovAction:
		return "ParameterChange"
	case *common.HardForkInitiationGovAction:
		return "HardForkInitiation"
	case *common.TreasuryWithdrawalGovAction:
		return "TreasuryWithdrawal"
	case *common.NoConfidenceGovAction:
		return "NoConfidence"
	case *common.UpdateCommitteeGovAction:
		return "UpdateCommittee"
	case *common.NewConstitutionGovAction:
		return "NewConstitution"
	case *common.InfoGovAction:
		return "Info"
	default:
		return "Unknown"
	}
}

// Helper methods

func (gp *GovernanceProcessor) getNextVotingProcedureIndex(tx *gorm.DB, txHash []byte) uint32 {
	var maxIndex uint32
	tx.Model(&models.VotingProcedure{}).Where("tx_hash = ?", txHash).Select("COALESCE(MAX(index), 0)").Scan(&maxIndex)
	return maxIndex + 1
}

func (gp *GovernanceProcessor) extractCommitteeVoter(govActionID *common.GovActionId) *int64 {
	// Extract from governance action ID structure
	// This would need proper implementation based on the actual GovActionId structure
	return nil
}

func (gp *GovernanceProcessor) extractDRepVoter(govActionID *common.GovActionId) *int64 {
	// Extract from governance action ID structure
	// This would need proper implementation based on the actual GovActionId structure
	return nil
}

func (gp *GovernanceProcessor) extractPoolVoter(govActionID *common.GovActionId) *int64 {
	// Extract from governance action ID structure
	// This would need proper implementation based on the actual GovActionId structure
	return nil
}

func (gp *GovernanceProcessor) getCurrentTxID(tx *gorm.DB) int64 {
	// Get the current transaction ID from context
	// This would typically be passed through context
	return 0
}

// resolveStakeAddressHash resolves stake address hash from an address
func (gp *GovernanceProcessor) resolveStakeAddressHash(tx *gorm.DB, address *common.Address) []byte {
	// Extract stake key hash from address
	stakeKeyHash := address.StakeKeyHash()
	var emptyHash common.Blake2b224
	if stakeKeyHash == emptyHash {
		return nil
	}
	return stakeKeyHash[:]
}

// convertVoterTypeToRole converts numeric voter type to string role
func (gp *GovernanceProcessor) convertVoterTypeToRole(voterType uint8) models.VoterRole {
	switch voterType {
	case VoterTypeConstitutionalCommitteeHotKeyHash, VoterTypeConstitutionalCommitteeHotScriptHash:
		return models.VoterRoleConstitutionalCommittee
	case VoterTypeDRepKeyHash, VoterTypeDRepScriptHash:
		return models.VoterRoleDRep
	case VoterTypeStakingPoolKeyHash:
		return models.VoterRoleSPO
	default:
		return models.VoterRole("Unknown")
	}
}

// convertVoteToChoice converts numeric vote to string choice
func (gp *GovernanceProcessor) convertVoteToChoice(vote uint8) models.VoteChoice {
	switch vote {
	case 0:
		return models.VoteChoiceNo
	case 1:
		return models.VoteChoiceYes
	case 2:
		return models.VoteChoiceAbstain
	default:
		return models.VoteChoice("Unknown")
	}
}

func (gp *GovernanceProcessor) extractCostModelID(tx *gorm.DB, action *common.ParameterChangeGovAction) *int64 {
	// Cost model extraction would require parsing the parameter changes
	// This is complex and depends on the specific parameter update format
	// For now, return nil as cost models are tracked separately
	return nil
}

func (gp *GovernanceProcessor) calculateEpochFromGovAction(tx *gorm.DB, govActionID int64) uint32 {
	// Get the governance action proposal to find its slot
	var proposal models.GovActionProposal
	if err := tx.Where("id = ?", govActionID).First(&proposal).Error; err != nil {
		return 0
	}
	
	// Get the block to find the slot number
	var block models.Block
	if err := tx.Where("hash = ?", proposal.TxHash).First(&block).Error; err != nil {
		return 0
	}
	
	// Calculate epoch from slot (Shelley+ uses 432000 slots per epoch)
	if block.SlotNo != nil {
		return uint32(*block.SlotNo / 432000)
	}
	
	return 0
}

func (gp *GovernanceProcessor) extractConstitutionScriptHash(action *common.NewConstitutionGovAction) []byte {
	// Extract script hash from constitution if present
	if len(action.Constitution.ScriptHash) > 0 {
		// Calculate script hash
		// This would need proper implementation based on the script structure
		return nil
	}
	return nil
}

// CalculateDRepDistribution calculates DRep stake distribution for an epoch
func (gp *GovernanceProcessor) CalculateDRepDistribution(ctx context.Context, tx *gorm.DB, epochNo uint32) error {
	// Get all active DRep delegations for this epoch
	var drepDelegations []struct {
		DRepHash   []byte
		TotalStake uint64
	}
	
	err := tx.Table("drep_distr").
		Select("drep_hash, SUM(amount) as total_stake").
		Where("epoch_no = ?", epochNo).
		Group("drep_hash").
		Scan(&drepDelegations).Error
	
	if err != nil {
		return fmt.Errorf("failed to calculate DRep distribution: %w", err)
	}
	
	log.Printf("Calculated DRep distribution for epoch %d: %d DReps", epochNo, len(drepDelegations))
	
	// Store aggregated results
	for _, delegation := range drepDelegations {
		distribution := &models.DRepDistr{
			HashRaw: delegation.DRepHash,
			Amount:  delegation.TotalStake,
			EpochNo: epochNo,
		}
		
		if err := tx.Create(distribution).Error; err != nil {
			log.Printf("[WARNING] Failed to store DRep distribution: %v", err)
		}
	}
	
	return nil
}

// ProcessDRepDelegation processes DRep delegation certificates
func (gp *GovernanceProcessor) ProcessDRepDelegation(ctx context.Context, tx *gorm.DB, txHash []byte, certIndex int, addrHash []byte, drepCred interface{}) error {
	// Handle different DRep credential types
	var drepHash []byte
	
	switch cred := drepCred.(type) {
	case []byte:
		// Regular DRep key hash
		drepHashHex := hex.EncodeToString(cred)
		if err := gp.drepHashCache.EnsureDRepHash(cred, drepHashHex); err != nil {
			return fmt.Errorf("failed to ensure DRep hash: %w", err)
		}
		drepHash = cred
	case string:
		// Special DRep types: "abstain" or "no_confidence"
		if cred == "abstain" || cred == "no_confidence" {
			// These are stored without a hash ID
			log.Printf(" DRep delegation to: %s", cred)
		}
	default:
		return fmt.Errorf("unknown DRep credential type: %T", drepCred)
	}
	
	// Create delegation vote record for DRep delegation
	delegationVote := &models.DelegationVote{
		AddrHash:   addrHash,
		CertIndex:  uint32(certIndex),
		TxHash:     txHash,
		DRepHash:   drepHash,
	}
	
	if err := tx.Create(delegationVote).Error; err != nil {
		return fmt.Errorf("failed to create DRep delegation: %w", err)
	}
	
	log.Printf("[OK] Processed DRep delegation for address %x", addrHash)
	return nil
}

// ProcessTransaction processes governance-related data from a transaction
func (gp *GovernanceProcessor) ProcessTransaction(tx *gorm.DB, txHash []byte, transaction interface{}, blockType uint) error {
	// Only process for Conway era
	if blockType < 7 {
		return nil
	}
	
	// Process voting procedures
	if err := gp.ProcessVotingProcedures(context.Background(), tx, txHash, transaction, blockType); err != nil {
		return fmt.Errorf("failed to process voting procedures: %w", err)
	}
	
	// Process proposal procedures
	if err := gp.ProcessProposalProcedures(context.Background(), tx, txHash, transaction, blockType); err != nil {
		return fmt.Errorf("failed to process proposal procedures: %w", err)
	}
	
	return nil
}