package database

import (
	"errors"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "broken pipe",
			err:      errors.New("write: broken pipe"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("read: connection reset by peer"),
			expected: true,
		},
		{
			name:     "TiDB write conflict",
			err:      errors.New("Error 9007: Write conflict"),
			expected: true,
		},
		{
			name:     "region unavailable",
			err:      errors.New("Region is unavailable"),
			expected: true,
		},
		{
			name:     "TiKV server busy",
			err:      errors.New("tikv is busy"),
			expected: true,
		},
		{
			name:     "deadlock",
			err:      errors.New("Error 1213: Deadlock found"),
			expected: true,
		},
		{
			name:     "lock wait timeout",
			err:      errors.New("Error 1205: Lock wait timeout exceeded"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("syntax error"),
			expected: false,
		},
		{
			name:     "permission denied",
			err:      errors.New("permission denied"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryOperation(t *testing.T) {
	t.Run("successful operation", func(t *testing.T) {
		callCount := 0
		err := RetryOperation(func() error {
			callCount++
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	t.Run("retryable error then success", func(t *testing.T) {
		callCount := 0
		err := RetryOperation(func() error {
			callCount++
			if callCount < 3 {
				return errors.New("connection reset by peer")
			}
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 3 {
			t.Errorf("Expected 3 calls, got %d", callCount)
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		callCount := 0
		expectedErr := errors.New("syntax error")
		err := RetryOperation(func() error {
			callCount++
			return expectedErr
		})

		if err != expectedErr {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		callCount := 0
		err := RetryOperation(func() error {
			callCount++
			return errors.New("connection reset by peer")
		})

		if err == nil {
			t.Error("Expected error after max retries")
		}
		if callCount != 4 { // 1 initial + 3 retries
			t.Errorf("Expected 4 calls, got %d", callCount)
		}
	})
}

func TestRetryTransaction(t *testing.T) {
	// Note: This is a unit test without actual database
	// In a real scenario, you'd use a test database or mock
	// Since we can't test with a real *gorm.DB, we'll skip these tests
	t.Skip("Skipping RetryTransaction tests - requires real database connection")
}

func TestExponentialBackoff(t *testing.T) {
	// Test that retry delays follow exponential backoff
	// We'll measure the actual sleep times between retries
	
	var attempts []time.Time
	err := RetryOperation(func() error {
		attempts = append(attempts, time.Now())
		if len(attempts) < 4 { // Force 3 retries
			return errors.New("connection reset")
		}
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected operation to eventually succeed, got: %v", err)
	}
	
	if len(attempts) != 4 {
		t.Errorf("Expected 4 attempts (1 initial + 3 retries), got %d", len(attempts))
	}
	
	// Check delays between attempts
	// First retry should be ~100ms, second ~200ms, third ~400ms
	expectedDelays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
	}
	
	for i := 1; i < len(attempts); i++ {
		actualDelay := attempts[i].Sub(attempts[i-1])
		expectedDelay := expectedDelays[i-1]
		
		// Allow 50ms tolerance for timing variations
		tolerance := 50 * time.Millisecond
		if actualDelay < expectedDelay-tolerance || actualDelay > expectedDelay+tolerance {
			t.Errorf("Retry %d: expected delay ~%v, got %v", i, expectedDelay, actualDelay)
		}
	}
}

func BenchmarkRetryOperation_Success(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RetryOperation(func() error {
			return nil
		})
	}
}

func BenchmarkRetryOperation_OneRetry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		callCount := 0
		RetryOperation(func() error {
			callCount++
			if callCount == 1 {
				return errors.New("connection reset")
			}
			return nil
		})
	}
}