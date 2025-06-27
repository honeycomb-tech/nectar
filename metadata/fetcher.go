// Copyright 2025 The Nectar Authors
// Off-chain metadata fetcher service for pool and governance metadata

package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	unifiederrors "nectar/errors"

	"golang.org/x/time/rate"
	"gorm.io/gorm"
	"nectar/models"
)

// Config holds configuration for the metadata fetcher
type Config struct {
	MaxRetries     int
	RetryBackoff   time.Duration
	RequestTimeout time.Duration
	RateLimit      rate.Limit
	RateBurst      int
	MaxContentSize int64
	UserAgent      string
	WorkerCount    int
	QueueSize      int
	FetchInterval  time.Duration
	AllowedSchemes []string
	MaxRedirects   int
	ValidateJSON   bool
}

// DefaultConfig returns sensible production defaults
func DefaultConfig() Config {
	return Config{
		MaxRetries:     3,
		RetryBackoff:   5 * time.Second,
		RequestTimeout: 30 * time.Second,
		RateLimit:      rate.Limit(10), // 10 requests per second
		RateBurst:      20,
		MaxContentSize: 10 * 1024 * 1024, // 10MB max
		UserAgent:      "Nectar/1.0 (Cardano Metadata Fetcher)",
		WorkerCount:    5,
		QueueSize:      1000,
		FetchInterval:  30 * time.Second,
		AllowedSchemes: []string{"https", "http"},
		MaxRedirects:   5,
		ValidateJSON:   true,
	}
}

// FetchRequest represents a metadata fetch request
type FetchRequest struct {
	ID           int64
	Type         string // "pool" or "governance"
	URL          string
	ExpectedHash []byte
	RetryCount   int
	Context      map[string]interface{} // Additional context (pool_id, anchor_id, etc)
}

// Fetcher handles off-chain metadata fetching
type Fetcher struct {
	config      Config
	db          *gorm.DB
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	queue       chan FetchRequest
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	processing  map[string]bool // Track URLs being processed
}

// New creates a new metadata fetcher
func New(db *gorm.DB, config Config) *Fetcher {
	ctx, cancel := context.WithCancel(context.Background())

	// Configure HTTP client with production settings
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &Fetcher{
		config:      config,
		db:          db,
		httpClient:  httpClient,
		rateLimiter: rate.NewLimiter(config.RateLimit, config.RateBurst),
		queue:       make(chan FetchRequest, config.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		processing:  make(map[string]bool),
	}
}

// Start begins the metadata fetching service
func (f *Fetcher) Start() error {
	log.Printf("Starting metadata fetcher service - workers=%d, queue_size=%d, rate_limit=%v",
		f.config.WorkerCount, f.config.QueueSize, f.config.RateLimit)

	// Start worker goroutines
	for i := 0; i < f.config.WorkerCount; i++ {
		f.wg.Add(1)
		go f.worker(i)
	}

	// Start queue processor - removed as not needed

	// Start periodic scanner for missed metadata
	f.wg.Add(1)
	go f.periodicScanner()

	return nil
}

// Stop gracefully shuts down the fetcher
func (f *Fetcher) Stop() error {
	log.Println("Stopping metadata fetcher service")
	f.cancel()
	close(f.queue)
	f.wg.Wait()
	return nil
}

// QueuePoolMetadata queues a pool metadata fetch request
func (f *Fetcher) QueuePoolMetadata(poolHash, pmrHash []byte, url string, hash []byte) error {
	// Generate a pseudo-ID from the hash for compatibility with existing code
	pmrID := hashToID(pmrHash)

	req := FetchRequest{
		ID:           pmrID,
		Type:         "pool",
		URL:          url,
		ExpectedHash: hash,
		Context: map[string]interface{}{
			"pool_hash": poolHash,
			"pmr_hash":  pmrHash,
			"pool_id":   hashToID(poolHash), // Keep for backward compatibility
			"pmr_id":    pmrID,
		},
	}

	select {
	case f.queue <- req:
		return nil
	case <-f.ctx.Done():
		return fmt.Errorf("fetcher is shutting down")
	default:
		return fmt.Errorf("queue is full")
	}
}

// QueueGovernanceMetadata queues a governance metadata fetch request
func (f *Fetcher) QueueGovernanceMetadata(anchorHash []byte, url string, hash []byte) error {
	// Generate a pseudo-ID from the hash for compatibility with existing code
	anchorID := hashToID(anchorHash)

	req := FetchRequest{
		ID:           anchorID,
		Type:         "governance",
		URL:          url,
		ExpectedHash: hash,
		Context: map[string]interface{}{
			"anchor_hash": anchorHash,
			"anchor_id":   anchorID, // Keep for backward compatibility
		},
	}

	select {
	case f.queue <- req:
		return nil
	case <-f.ctx.Done():
		return fmt.Errorf("fetcher is shutting down")
	default:
		return fmt.Errorf("queue is full")
	}
}

// worker processes fetch requests from the queue
func (f *Fetcher) worker(id int) {
	defer f.wg.Done()
	// Worker started (debug logging removed)

	for {
		select {
		case req, ok := <-f.queue:
			if !ok {
				// Worker stopping, queue closed
				return
			}

			// Check if URL is already being processed
			if f.isProcessing(req.URL) {
				// Re-queue for later
				go func() {
					time.Sleep(5 * time.Second)
					f.queue <- req
				}()
				continue
			}

			// Mark as processing
			f.setProcessing(req.URL, true)
			// Process the request
			err := f.processFetchRequest(req)

			// Unmark as processing
			f.setProcessing(req.URL, false)

			if err != nil {
				// Only log metadata fetch errors in debug mode to reduce noise
				// These are expected for old/expired pool metadata URLs
				if req.Type == "pool" && (strings.Contains(err.Error(), "404") || 
					strings.Contains(err.Error(), "no such host") || 
					strings.Contains(err.Error(), "hash mismatch")) {
					// Common expected errors - don't log unless in debug mode
					if f.config.ValidateJSON { // Using ValidateJSON as a proxy for debug mode
						log.Printf("[DEBUG] Pool metadata fetch failed (expected): %s - %v", req.URL, err)
					}
				} else {
					// Log other errors to unified error system
					unifiederrors.Get().LogError(
						unifiederrors.ErrorTypeNetwork,
						"MetadataFetcher",
						"Error",
						fmt.Sprintf("Failed to fetch %s metadata from %s: %v", req.Type, req.URL, err),
						fmt.Sprintf("retry_count=%d", req.RetryCount),
					)
				}

				// Retry logic
				if req.RetryCount < f.config.MaxRetries {
					req.RetryCount++
					go func() {
						backoff := f.config.RetryBackoff * time.Duration(req.RetryCount)
						time.Sleep(backoff)
						f.queue <- req
					}()
				} else {
					// Max retries reached, record permanent failure
					f.recordFetchError(req, err)
				}
			} else {
			}

		case <-f.ctx.Done():
			// Worker stopping, context cancelled
			return
		}
	}
}

// processFetchRequest handles a single fetch request
func (f *Fetcher) processFetchRequest(req FetchRequest) error {
	// Validate URL
	parsedURL, err := f.validateURL(req.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Rate limiting
	if err := f.rateLimiter.Wait(f.ctx); err != nil {
		return fmt.Errorf("rate limiter error: %w", err)
	}

	// Fetch metadata
	data, err := f.fetchURL(parsedURL)
	if err != nil {
		return fmt.Errorf("fetch error: %w", err)
	}

	// Verify hash
	if !f.verifyHash(data, req.ExpectedHash) {
		return fmt.Errorf("hash mismatch")
	}

	// Parse and validate JSON
	var jsonData interface{}
	if f.config.ValidateJSON {
		if err := json.Unmarshal(data, &jsonData); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	}

	// Store metadata based on type
	switch req.Type {
	case "pool":
		return f.storePoolMetadata(req, data, jsonData)
	case "governance":
		return f.storeGovernanceMetadata(req, data, jsonData)
	default:
		return fmt.Errorf("unknown metadata type: %s", req.Type)
	}
}

// validateURL validates and parses the URL
func (f *Fetcher) validateURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Check allowed schemes
	allowed := false
	for _, scheme := range f.config.AllowedSchemes {
		if parsedURL.Scheme == scheme {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("scheme not allowed: %s", parsedURL.Scheme)
	}

	// Additional validation
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("missing host")
	}

	return parsedURL, nil
}

// fetchURL fetches content from a URL with size limits
func (f *Fetcher) fetchURL(parsedURL *url.URL) ([]byte, error) {
	req, err := http.NewRequestWithContext(f.ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", f.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, f.config.MaxContentSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// verifyHash verifies the content hash
func (f *Fetcher) verifyHash(data []byte, expectedHash []byte) bool {
	actualHash := sha256.Sum256(data)
	return hex.EncodeToString(actualHash[:]) == hex.EncodeToString(expectedHash)
}

// storePoolMetadata stores fetched pool metadata
func (f *Fetcher) storePoolMetadata(req FetchRequest, data []byte, jsonData interface{}) error {
	// poolID := req.Context["pool_id"].(uint64) // Not used with hash-based model
	// pmrID := req.Context["pmr_id"].(uint64) // Not used with hash-based model

	// Parse pool metadata structure
	var poolMeta struct {
		Name        string `json:"name"`
		Ticker      string `json:"ticker"`
		Homepage    string `json:"homepage"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(data, &poolMeta); err != nil {
		return fmt.Errorf("failed to parse pool metadata: %w", err)
	}

	// Validate ticker length
	if len(poolMeta.Ticker) > 5 {
		poolMeta.Ticker = poolMeta.Ticker[:5]
	}

	// Get pool and pmr hashes from context
	poolHash := req.Context["pool_hash"].([]byte)
	pmrHash := req.Context["pmr_hash"].([]byte)

	// Store in database
	offChainData := &models.OffChainPoolData{
		Hash:       req.ExpectedHash,
		PoolHash:   poolHash,
		TickerName: poolMeta.Ticker,
		Json:       string(data),
		Bytes:      data,
		PmrHash:    pmrHash,
	}

	if err := f.db.Create(offChainData).Error; err != nil {
		return fmt.Errorf("failed to store pool metadata: %w", err)
	}

	// Successfully stored pool metadata

	return nil
}

// storeGovernanceMetadata stores fetched governance metadata
func (f *Fetcher) storeGovernanceMetadata(req FetchRequest, data []byte, jsonData interface{}) error {
	// anchorID := req.Context["anchor_id"].(uint64) // Not used with hash-based model
	anchorHash := req.Context["anchor_hash"].([]byte)

	// Parse CIP-119 JSON-LD structure
	var govMeta map[string]interface{}
	if err := json.Unmarshal(data, &govMeta); err != nil {
		return fmt.Errorf("failed to parse governance metadata: %w", err)
	}

	// Extract common fields
	language, _ := govMeta["@language"].(string)
	comment, _ := govMeta["comment"].(string)

	// Validate and determine type
	isValid := f.validateGovernanceMetadata(govMeta)
	var warning string
	if !isValid {
		warning = "Invalid CIP-119 format"
	}

	// Get anchor hash from context (already declared above)
	// anchorHash := req.Context["anchor_hash"].([]byte)

	// Store main vote data
	voteData := &models.OffChainVoteData{
		Hash:             req.ExpectedHash,
		VotingAnchorHash: anchorHash,
		Language:         language,
		Comment:          &comment,
		Json:             string(data),
		Bytes:            data,
		Warning:          &warning,
		IsValid:          isValid,
	}

	tx := f.db.Begin()
	if err := tx.Create(voteData).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to store vote data: %w", err)
	}

	// Store additional governance data based on type
	govMeta["anchorHash"] = anchorHash      // Pass anchorHash through meta
	govMeta["voteDataHash"] = voteData.Hash // Pass vote data hash
	if err := f.storeGovernanceDetails(tx, voteData.Hash, govMeta); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to store governance details: %w", err)
	}

	tx.Commit()

	// Successfully stored governance metadata

	return nil
}

// storeGovernanceDetails stores type-specific governance metadata
func (f *Fetcher) storeGovernanceDetails(tx *gorm.DB, voteDataHash []byte, meta map[string]interface{}) error {
	metaType, _ := meta["@type"].(string)
	anchorHash := meta["anchorHash"].([]byte) // Pass through from caller

	switch metaType {
	case "GovernanceAction":
		// Store governance action data
		title, _ := meta["title"].(string)
		abstract, _ := meta["abstract"].(string)
		motivation, _ := meta["motivation"].(string)
		rationale, _ := meta["rationale"].(string)
		language, _ := meta["@language"].(string)

		govAction := &models.OffChainVoteGovActionData{
			OffChainVoteDataHash: voteDataHash,
			VotingAnchorHash:     anchorHash,
			Language:             language,
			Title:                &title,
			Abstract:             &abstract,
			Motivation:           &motivation,
			Rationale:            &rationale,
		}
		if err := tx.Create(govAction).Error; err != nil {
			return err
		}

	case "DRepMetadata":
		// Store DRep metadata
		bio, _ := meta["bio"].(string)
		email, _ := meta["email"].(string)
		givenName, _ := meta["givenName"].(string)
		language, _ := meta["@language"].(string)
		comment, _ := meta["comment"].(string)
		paymentAddress, _ := meta["paymentAddress"].(string)
		image, _ := meta["image"].(string)
		objectives, _ := meta["objectives"].(string)
		motivations, _ := meta["motivations"].(string)
		qualifications, _ := meta["qualifications"].(string)
		doNotList, _ := meta["doNotList"].(bool)

		// TODO: Get actual DRep hash from context
		drepHashRaw := []byte{} // This should come from the transaction context

		drepData := &models.OffChainVoteDRepData{
			OffChainVoteDataHash: voteDataHash,
			DRepHashRaw:          drepHashRaw,
			VotingAnchorHash:     anchorHash,
			Language:             language,
			Comment:              &comment,
			Bio:                  &bio,
			Email:                &email,
			PaymentAddress:       &paymentAddress,
			GivenName:            &givenName,
			Image:                &image,
			Objectives:           &objectives,
			Motivations:          &motivations,
			Qualifications:       &qualifications,
			DoNotList:            doNotList,
		}
		if err := tx.Create(drepData).Error; err != nil {
			return err
		}

	default:
		// Store as generic vote data
		// Unknown governance metadata type
	}

	// Store authors if present
	if authors, ok := meta["authors"].([]interface{}); ok {
		for _, author := range authors {
			if authorMap, ok := author.(map[string]interface{}); ok {
				name := authorMap["name"].(string)
				var witnessBytes []byte
				if witnessData, ok := authorMap["witness"].(string); ok {
					// Assuming witness is a hex-encoded string
					if decoded, err := hex.DecodeString(witnessData); err == nil {
						witnessBytes = decoded
					}
				}
				// Generate hash for author
				authorHash := models.GenerateOffChainVoteAuthorHash(voteDataHash, &name, witnessBytes)

				authorData := &models.OffChainVoteAuthor{
					Hash:                 authorHash,
					OffChainVoteDataHash: voteDataHash,
					Name:                 &name,
					Witness:              witnessBytes,
				}
				if err := tx.Create(authorData).Error; err != nil {
					return err
				}
			}
		}
	}

	// Store references if present
	if refs, ok := meta["references"].([]interface{}); ok {
		for _, ref := range refs {
			if refMap, ok := ref.(map[string]interface{}); ok {
				label := refMap["label"].(string)
				uri := refMap["uri"].(string)
				var referenceHash []byte
				if hashStr, ok := refMap["hashDigest"].(string); ok {
					// Decode hex-encoded hash
					if decoded, err := hex.DecodeString(hashStr); err == nil {
						referenceHash = decoded
					}
				}
				refData := &models.OffChainVoteReference{
					OffChainVoteDataHash: voteDataHash,
					Label:                label,
					URI:                  uri,
					ReferenceHash:        referenceHash,
				}
				if err := tx.Create(refData).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validateGovernanceMetadata validates CIP-119 format
func (f *Fetcher) validateGovernanceMetadata(meta map[string]interface{}) bool {
	// Check required fields
	requiredFields := []string{"@context", "@type"}
	for _, field := range requiredFields {
		if _, ok := meta[field]; !ok {
			return false
		}
	}

	// Validate context
	context, ok := meta["@context"].(map[string]interface{})
	if !ok {
		return false
	}

	// Check for CIP-119 context
	cip119, ok := context["CIP119"].(string)
	if !ok || cip119 != "https://github.com/cardano-foundation/CIPs/blob/master/CIP-0119/CIP-0119.jsonld" {
		return false
	}

	return true
}

// recordFetchError records a permanent fetch failure
func (f *Fetcher) recordFetchError(req FetchRequest, err error) {
	switch req.Type {
	case "pool":
		poolHash := req.Context["pool_hash"].([]byte)
		pmrHash := req.Context["pmr_hash"].([]byte)

		fetchError := &models.OffChainPoolFetchError{
			PoolHash:   poolHash,
			PmrHash:    pmrHash,
			FetchTime:  time.Now(),
			FetchError: err.Error(),
			RetryCount: uint32(req.RetryCount),
		}

		if dbErr := f.db.Create(fetchError).Error; dbErr != nil {
			unifiederrors.Get().DatabaseError("MetadataFetcher", "RecordPoolError", dbErr)
		}

	case "governance":
		anchorHash := req.Context["anchor_hash"].([]byte)

		fetchError := &models.OffChainVoteFetchError{
			VotingAnchorHash: anchorHash,
			FetchTime:        time.Now(),
			FetchError:       err.Error(),
			RetryCount:       uint32(req.RetryCount),
		}

		if dbErr := f.db.Create(fetchError).Error; dbErr != nil {
			unifiederrors.Get().DatabaseError("MetadataFetcher", "RecordGovernanceError", dbErr)
		}
	}
}

// periodicScanner scans for unfetched metadata periodically
func (f *Fetcher) periodicScanner() {
	defer f.wg.Done()
	ticker := time.NewTicker(f.config.FetchInterval)
	defer ticker.Stop()
	
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ticker.C:
			f.scanUnfetchedMetadata()
		case <-cleanupTicker.C:
			f.cleanupProcessingMap()
		case <-f.ctx.Done():
			return
		}
	}
}

// scanUnfetchedMetadata finds and queues unfetched metadata
func (f *Fetcher) scanUnfetchedMetadata() {
	// Scan for unfetched pool metadata
	var unfetchedPools []struct {
		PoolHash []byte
		PmrHash  []byte
		URL      string
		Hash     []byte
	}

	query := `
		SELECT pmr.pool_hash, pmr.hash as pmr_hash, pmr.url, pmr.hash
		FROM pool_metadata_refs pmr
		LEFT JOIN off_chain_pool_data ocpd ON pmr.hash = ocpd.pmr_hash
		LEFT JOIN off_chain_pool_fetch_errors ocpfe ON pmr.hash = ocpfe.pmr_hash
		WHERE ocpd.pmr_hash IS NULL 
		AND (ocpfe.pmr_hash IS NULL OR ocpfe.retry_count < ?)
		LIMIT 100
	`

	if err := f.db.Raw(query, f.config.MaxRetries).Scan(&unfetchedPools).Error; err != nil {
		unifiederrors.Get().DatabaseError("MetadataFetcher", "ScanPoolMetadata", err)
		return
	}

	for _, pool := range unfetchedPools {
		if err := f.QueuePoolMetadata(pool.PoolHash, pool.PmrHash, pool.URL, pool.Hash); err != nil {
			// Failed to queue pool metadata
		}
	}

	// Scan for unfetched governance metadata
	var unfetchedAnchors []struct {
		AnchorHash []byte
		URL        string
		Hash       []byte
	}

	query = `
		SELECT va.hash as anchor_hash, va.url, va.data_hash as hash
		FROM voting_anchors va
		LEFT JOIN off_chain_vote_data ocvd ON va.hash = ocvd.voting_anchor_hash
		LEFT JOIN off_chain_vote_fetch_errors ocvfe ON va.hash = ocvfe.voting_anchor_hash
		WHERE ocvd.voting_anchor_hash IS NULL
		AND (ocvfe.voting_anchor_hash IS NULL OR ocvfe.retry_count < ?)
		LIMIT 100
	`

	if err := f.db.Raw(query, f.config.MaxRetries).Scan(&unfetchedAnchors).Error; err != nil {
		unifiederrors.Get().DatabaseError("MetadataFetcher", "ScanGovernanceMetadata", err)
		return
	}

	for _, anchor := range unfetchedAnchors {
		if err := f.QueueGovernanceMetadata(anchor.AnchorHash, anchor.URL, anchor.Hash); err != nil {
			// Failed to queue governance metadata
		}
	}

	// Periodic metadata scan complete
}

// Helper methods for concurrency control
func (f *Fetcher) isProcessing(url string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.processing[url]
}

func (f *Fetcher) setProcessing(url string, status bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if status {
		f.processing[url] = true
	} else {
		delete(f.processing, url)
	}
}

// Simple logger implementation using standard log package
type simpleLogger struct{}

func (s simpleLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("[INFO] %s %v", msg, keysAndValues)
}

func (s simpleLogger) Debug(msg string, keysAndValues ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] %s %v", msg, keysAndValues)
	}
}

func (s simpleLogger) Error(msg string, keysAndValues ...interface{}) {
	unifiederrors.Get().LogError(unifiederrors.ErrorTypeNetwork, "MetadataFetcher", "Error", fmt.Sprintf("%s %v", msg, keysAndValues))
}

// Package-level logger
var (
	logger    = simpleLogger{}
	debugMode = false
)

// hashToID converts a hash to a pseudo-ID for compatibility
// This is a temporary solution for backward compatibility with existing code
func hashToID(hash []byte) int64 {
	if len(hash) == 0 {
		return 0
	}

	// Use first 8 bytes of hash as a pseudo-ID
	var id int64
	for i := 0; i < 8 && i < len(hash); i++ {
		id = (id << 8) | int64(hash[i])
	}

	// Ensure positive
	if id < 0 {
		id = -id
	}

	return id
}

// cleanupProcessingMap removes stale entries from the processing map
func (f *Fetcher) cleanupProcessingMap() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Clear the entire processing map periodically
	// This prevents memory leaks from URLs that never complete
	oldSize := len(f.processing)
	f.processing = make(map[string]bool)
	
	if oldSize > 0 {
		log.Printf("[MetadataFetcher] Cleaned up %d stale processing entries", oldSize)
	}
}
