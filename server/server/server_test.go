package server

import (
	"testing"
)

func TestCheckWinnerDraw(t *testing.T) {
	board := [3][3]string{
		{"X", "O", "X"},
		{"X", "O", "O"},
		{"O", "X", "X"},
	}
	winner, draw := CheckWinner(board)
	if winner != "" || !draw {
		t.Errorf("Expected draw, got winner=%q draw=%v", winner, draw)
	}
}

