package processors

import "sync/atomic"

// LoggingConfig controls verbosity of processor logging
type LoggingConfig struct {
	// Certificate logging
	LogCertificateProcessing atomic.Bool
	LogCertificateDetails    atomic.Bool
	
	// Block logging
	LogBlockProcessing       atomic.Bool
	LogTransactionProcessing atomic.Bool
	
	// Staking logging
	LogStakeOperations atomic.Bool
	LogPoolOperations  atomic.Bool
}

// GlobalLoggingConfig is the global logging configuration
var GlobalLoggingConfig = &LoggingConfig{}

// InitLoggingConfig initializes the logging configuration
func InitLoggingConfig(verbose bool) {
	// For production, turn off verbose certificate logging
	GlobalLoggingConfig.LogCertificateProcessing.Store(false)
	GlobalLoggingConfig.LogCertificateDetails.Store(false)
	GlobalLoggingConfig.LogStakeOperations.Store(false)
	GlobalLoggingConfig.LogPoolOperations.Store(false)
	
	// Show some essential progress info
	GlobalLoggingConfig.LogBlockProcessing.Store(true)
	GlobalLoggingConfig.LogTransactionProcessing.Store(false)
	
	// If verbose mode requested, enable all
	if verbose {
		GlobalLoggingConfig.LogCertificateProcessing.Store(true)
		GlobalLoggingConfig.LogCertificateDetails.Store(true)
		GlobalLoggingConfig.LogStakeOperations.Store(true)
		GlobalLoggingConfig.LogPoolOperations.Store(true)
		GlobalLoggingConfig.LogTransactionProcessing.Store(true)
	}
}