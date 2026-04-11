package domain

// ScoreUpdater atualiza o score dos usuários após o fechamento de uma rodada.
// Implementação real será feita em feature/score-badges.
type ScoreUpdater interface {
	UpdateAfterRound(roundID string) error
}

// NoopScoreUpdater é a implementação stub usada até feature/score-badges.
type NoopScoreUpdater struct{}

func (n *NoopScoreUpdater) UpdateAfterRound(roundID string) error { return nil }
