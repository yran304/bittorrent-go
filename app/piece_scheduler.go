package main

import (
	"fmt"
	"sync"
)

type pieceStatus int

const (
	piecePending pieceStatus = iota
	pieceInProgress
	pieceComplete
)

type scheduledPiece struct {
	status   pieceStatus
	attempts int
}

type pieceScheduler struct {
	mu             sync.Mutex
	pieces         []scheduledPiece
	completedCount int
}

func newPieceScheduler(pieceCount int) *pieceScheduler {
	return &pieceScheduler{
		pieces: make([]scheduledPiece, pieceCount),
	}
}

func (s *pieceScheduler) nextPieceForPeer(bitfield []byte) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for pieceIndex, piece := range s.pieces {
		if piece.status != piecePending {
			continue
		}

		if !hasPiece(bitfield, pieceIndex) {
			continue
		}

		s.pieces[pieceIndex].status = pieceInProgress
		s.pieces[pieceIndex].attempts++
		return pieceIndex, true
	}

	return 0, false
}

func (s *pieceScheduler) markComplete(pieceIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pieces[pieceIndex].status != pieceInProgress {
		return fmt.Errorf("cannot mark piece %d complete from status %d", pieceIndex, s.pieces[pieceIndex].status)
	}

	s.pieces[pieceIndex].status = pieceComplete
	s.completedCount++
	return nil
}

func (s *pieceScheduler) markAttemptFailed(pieceIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// markAttemptFailed returns an error when this piece has no attempts left.
	if s.pieces[pieceIndex].attempts >= maxPieceAttempts {
		return fmt.Errorf("piece %d failed after %d download attempts", pieceIndex, maxPieceAttempts)
	}

	s.pieces[pieceIndex].status = piecePending
	return nil
}

func (s *pieceScheduler) isComplete() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.completedCount == len(s.pieces)
}
