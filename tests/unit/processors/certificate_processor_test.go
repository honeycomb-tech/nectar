package processors

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"nectar/database"
)

// MockCertificate implements ledger.Certificate
type MockCertificate struct {
	mock.Mock
	certType ledger.CertificateType
}

func (m *MockCertificate) Type() ledger.CertificateType {
	return m.certType
}

func (m *MockCertificate) Cbor() []byte {
	return []byte{}
}

func (m *MockCertificate) Utxorpc() *any {
	return nil
}

// Mock specific certificate types
type MockStakeRegistrationCertificate struct {
	MockCertificate
	stakeCredential ledger.StakeCredential
}

func (m *MockStakeRegistrationCertificate) StakeCredential() ledger.StakeCredential {
	return m.stakeCredential
}

func (m *MockStakeRegistrationCertificate) Deposit() uint64 {
	return 2000000
}

type MockStakeDelegationCertificate struct {
	MockCertificate
	stakeCredential ledger.StakeCredential
	poolKeyHash     ledger.Blake2b224
}

func (m *MockStakeDelegationCertificate) StakeCredential() ledger.StakeCredential {
	return m.stakeCredential
}

func (m *MockStakeDelegationCertificate) PoolKeyHash() ledger.Blake2b224 {
	return m.poolKeyHash
}

type MockPoolRegistrationCertificate struct {
	MockCertificate
	operator ledger.Blake2b224
	pledge   uint64
	cost     uint64
	margin   ledger.UnitInterval
}

func (m *MockPoolRegistrationCertificate) Operator() ledger.Blake2b224 {
	return m.operator
}

func (m *MockPoolRegistrationCertificate) VrfKeyHash() ledger.Blake2b256 {
	return ledger.Blake2b256{}
}

func (m *MockPoolRegistrationCertificate) Pledge() uint64 {
	return m.pledge
}

func (m *MockPoolRegistrationCertificate) Cost() uint64 {
	return m.cost
}

func (m *MockPoolRegistrationCertificate) Margin() ledger.UnitInterval {
	return m.margin
}

func (m *MockPoolRegistrationCertificate) RewardAccount() ledger.Address {
	return ledger.Address{}
}

func (m *MockPoolRegistrationCertificate) PoolOwners() []ledger.Blake2b224 {
	return []ledger.Blake2b224{}
}

func (m *MockPoolRegistrationCertificate) Relays() []ledger.Relay {
	return []ledger.Relay{}
}

func (m *MockPoolRegistrationCertificate) PoolMetadata() *ledger.PoolMetadata {
	return nil
}

func setupTestDBForCert(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&database.StakeRegistration{},
		&database.StakeDeregistration{},
		&database.PoolRegistration{},
		&database.PoolRetirement{},
		&database.Delegation{},
		&database.DRepRegistration{},
		&database.DRepDeregistration{},
		&database.DRepUpdate{},
		&database.CommitteeHotKeyRegistration{},
		&database.CommitteeHotKeyDeregistration{},
		&database.CommitteeResignation{},
	)
	assert.NoError(t, err)

	return db
}

func TestNewCertificateProcessor(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.db)
	assert.NotNil(t, processor.stakeCache)
}

func TestProcessCertificate_StakeRegistration(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	cert := &MockStakeRegistrationCertificate{
		MockCertificate: MockCertificate{
			certType: ledger.CertificateTypeStakeRegistration,
		},
		stakeCredential: ledger.StakeCredential{},
	}

	err := processor.ProcessCertificate(nil, "test_tx_hash", 0, cert, 1, 100)
	assert.NoError(t, err)

	// Verify registration was saved
	var count int64
	db.Model(&database.StakeRegistration{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestProcessCertificate_StakeDelegation(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	cert := &MockStakeDelegationCertificate{
		MockCertificate: MockCertificate{
			certType: ledger.CertificateTypeStakeDelegation,
		},
		stakeCredential: ledger.StakeCredential{},
		poolKeyHash:     ledger.Blake2b224{},
	}

	err := processor.ProcessCertificate(nil, "test_tx_hash", 0, cert, 1, 100)
	assert.NoError(t, err)

	// Verify delegation was saved
	var count int64
	db.Model(&database.Delegation{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestProcessCertificate_PoolRegistration(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	cert := &MockPoolRegistrationCertificate{
		MockCertificate: MockCertificate{
			certType: ledger.CertificateTypePoolRegistration,
		},
		operator: ledger.Blake2b224{},
		pledge:   1000000000,
		cost:     340000000,
		margin:   ledger.UnitInterval{Numerator: 1, Denominator: 100},
	}

	err := processor.ProcessCertificate(nil, "test_tx_hash", 0, cert, 1, 100)
	assert.NoError(t, err)

	// Verify pool registration was saved
	var pool database.PoolRegistration
	err = db.First(&pool).Error
	assert.NoError(t, err)
	assert.Equal(t, uint64(1000000000), pool.Pledge)
	assert.Equal(t, uint64(340000000), pool.FixedCost)
}

func TestProcessCertificate_UnknownType(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	cert := &MockCertificate{
		certType: 999, // Unknown type
	}

	err := processor.ProcessCertificate(nil, "test_tx_hash", 0, cert, 1, 100)
	assert.NoError(t, err) // Should not error on unknown types
}

func TestProcessCertificate_BatchProcessing(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	// Create multiple certificates
	certs := []ledger.Certificate{
		&MockStakeRegistrationCertificate{
			MockCertificate: MockCertificate{certType: ledger.CertificateTypeStakeRegistration},
		},
		&MockStakeDelegationCertificate{
			MockCertificate: MockCertificate{certType: ledger.CertificateTypeStakeDelegation},
		},
		&MockPoolRegistrationCertificate{
			MockCertificate: MockCertificate{certType: ledger.CertificateTypePoolRegistration},
		},
	}

	// Process all certificates
	for i, cert := range certs {
		err := processor.ProcessCertificate(nil, "batch_tx_hash", uint32(i), cert, 1, 100)
		assert.NoError(t, err)
	}

	// Verify all were saved
	var stakeRegCount, delegationCount, poolRegCount int64
	db.Model(&database.StakeRegistration{}).Count(&stakeRegCount)
	db.Model(&database.Delegation{}).Count(&delegationCount)
	db.Model(&database.PoolRegistration{}).Count(&poolRegCount)

	assert.Equal(t, int64(1), stakeRegCount)
	assert.Equal(t, int64(1), delegationCount)
	assert.Equal(t, int64(1), poolRegCount)
}

func TestProcessCertificate_ErrorHandling(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	// Close database to force error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	cert := &MockStakeRegistrationCertificate{
		MockCertificate: MockCertificate{
			certType: ledger.CertificateTypeStakeRegistration,
		},
	}

	err := processor.ProcessCertificate(nil, "error_tx_hash", 0, cert, 1, 100)
	assert.Error(t, err)
}

func TestStakeAddressCaching(t *testing.T) {
	db := setupTestDBForCert(t)
	cache := NewStakeAddressCache(100)
	processor := NewCertificateProcessor(db, cache)

	// Test cache operations
	stakeAddr := "stake1234567890"
	stakeID := uint(123)

	// Add to cache
	processor.stakeCache.Set(stakeAddr, stakeID)

	// Retrieve from cache
	cachedID, found := processor.stakeCache.Get(stakeAddr)
	assert.True(t, found)
	assert.Equal(t, stakeID, cachedID)
}