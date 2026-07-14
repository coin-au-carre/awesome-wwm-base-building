package discord

import "testing"

func TestComputeScoreAnyEmoji(t *testing.T) {
	reactions := map[string][]string{"😂": {"u1"}}
	weights := map[string]float64{"u1": 1}

	if got := computeScore(reactions, weights, nil, "", ""); got != 0 {
		t.Errorf("computeScore should ignore unscored emoji, got %d", got)
	}
	if got := computeScoreWithOpts(reactions, weights, nil, "", "", true); got != scorePerLike {
		t.Errorf("computeScoreWithOpts(countAnyEmoji=true) = %d, want %d", got, scorePerLike)
	}
}
