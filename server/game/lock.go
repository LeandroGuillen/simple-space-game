package game

import "sync"

// Locking convention for Game.mu (*sync.RWMutex):
//
//   - Public entry points that mutate game state take the write lock (Lock).
//   - Public entry points that only observe state take the read lock (RLock).
//   - Methods invoked while a lock is already held MUST NOT lock again —
//     Go mutexes are not reentrant. Use the *Unlocked helpers from those
//     call sites (e.g. SubmitTurn → AdvanceTurn → exploreStarUnlocked).
//   - GetVisibleGameForPlayer builds a deep copy under RLock and never
//     mutates the live Game (command-point hydration runs on the copy only).
//   - BroadcastGameUpdate / websocket reads call GetVisibleGameForPlayer
//     after SubmitTurn has released the write lock, avoiding re-entrancy.
func (g *Game) ensureMu() {
	if g.mu == nil {
		g.mu = &sync.RWMutex{}
	}
}
