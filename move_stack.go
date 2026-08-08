package lacrima

type moveStack struct {
	moves []Move
}

func (stack *moveStack) push(move Move) {
	stack.moves = append(stack.moves, move)
}

func (stack *moveStack) pop() {
	stack.moves = stack.moves[:len(stack.moves)-1]
}

func (stack *moveStack) previous(pliesBack int) (Move, bool) {
	index := len(stack.moves) - 1 - pliesBack
	if pliesBack < 0 || index < 0 {
		return Move{}, false
	}
	return stack.moves[index], true
}

func (stack *moveStack) len() int {
	return len(stack.moves)
}
