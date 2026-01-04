// Package filelock provides file locking for concurrent access safety.
//
// FileLock uses advisory locking (syscall.Flock) to protect files during
// read-modify-write operations. Lock files are created with .lock extension
// alongside the target file.
//
// Usage:
//
//	lock, err := AcquireLock(filepath, 5*time.Second)
//	if err != nil {
//		return err
//	}
//	defer lock.Release()
//	// Safely read/modify/write the file
package filelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// FileLock represents an exclusive lock on a file.
type FileLock struct {
	file     *os.File
	lockPath string
}

// AcquireLock attempts to acquire an exclusive lock on the given path with a timeout.
// Returns ErrLockTimeout if the lock cannot be acquired within the timeout period.
// The lock file is created with .lock extension alongside the target file.
func AcquireLock(path string, timeout time.Duration) (*FileLock, error) {
	lockPath := path + ".lock"

	// Open or create the lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	// Try to acquire the lock with timeout
	deadline := time.Now().Add(timeout)
	for {
		// Try to acquire exclusive lock (non-blocking)
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Lock acquired successfully
			return &FileLock{
				file:     file,
				lockPath: lockPath,
			}, nil
		}

		// Check if it's not a "would block" error (indicates lock is held)
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			file.Close()
			return nil, fmt.Errorf("locking file: %w", err)
		}

		// Lock is held by another process, check timeout
		if time.Now().After(deadline) {
			file.Close()
			return nil, ErrLockTimeout
		}

		// Wait a short time before retrying
		time.Sleep(10 * time.Millisecond)
	}
}

// TryLock attempts a non-blocking lock acquisition.
// Returns ErrLockHeld if the lock cannot be acquired immediately.
func TryLock(path string) (*FileLock, error) {
	lockPath := path + ".lock"

	// Open or create the lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	// Try non-blocking exclusive lock
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLockHeld
		}
		return nil, fmt.Errorf("locking file: %w", err)
	}

	return &FileLock{
		file:     file,
		lockPath: lockPath,
	}, nil
}

// Release releases the lock and removes the lock file.
// Should be called in a defer statement to ensure cleanup.
func (l *FileLock) Release() error {
	if l.file == nil {
		return nil
	}

	// Unlock the file
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)

	// Close the file
	if err := l.file.Close(); err != nil {
		// Log but don't fail on close error
		return fmt.Errorf("closing lock file: %w", err)
	}

	// Clean up the lock file (best effort)
	_ = os.Remove(l.lockPath)

	l.file = nil
	return nil
}

// Errors returned by filelock operations
var (
	// ErrLockTimeout is returned when lock acquisition times out
	ErrLockTimeout = errors.New("lock acquisition timeout")

	// ErrLockHeld is returned by TryLock when the lock is immediately unavailable
	ErrLockHeld = errors.New("lock is held by another process")
)
