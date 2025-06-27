package constants

// Era slot boundaries
const (
	ByronEraEndSlot    = 4492799
	ShelleyEraEndSlot  = 16588799
	AllegraEraEndSlot  = 23068799
	MaryEraEndSlot     = 39916799
	AlonzoEraEndSlot   = 72316799
	BabbageEraEndSlot  = 999999999 // Ongoing
)

// Era names
const (
	ByronEra   = "Byron"
	ShelleyEra = "Shelley"
	AllegraEra = "Allegra"
	MaryEra    = "Mary"
	AlonzoEra  = "Alonzo"
	BabbageEra = "Babbage"
)

// Processing limits
const (
	DefaultBatchSize      = 100
	DefaultWorkerCount    = 10
	MaxRetries           = 3
	RetryDelay           = 5 // seconds
	CacheMaxEntries      = 10000
	ChannelBufferSize    = 1000
	ShutdownTimeout      = 30 // seconds
)

// Database constants
const (
	MaxConnectionRetries = 5
	ConnectionTimeout    = 30 // seconds
	QueryTimeout        = 60 // seconds
)

// GetEraFromSlot returns the era name for a given slot number
func GetEraFromSlot(slot uint64) string {
	switch {
	case slot <= ByronEraEndSlot:
		return ByronEra
	case slot <= ShelleyEraEndSlot:
		return ShelleyEra
	case slot <= AllegraEraEndSlot:
		return AllegraEra
	case slot <= MaryEraEndSlot:
		return MaryEra
	case slot <= AlonzoEraEndSlot:
		return AlonzoEra
	default:
		return BabbageEra
	}
}