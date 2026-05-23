# Lacrima
## An engine so bad it'll make you cry.

- written in Go
- should be better structured than ethantron (my old python engine)
DONE:
  - movegen, board rep, legality filtering
  - PESTO eval, negamax with a/b pruning
  - move ordering based on mvv/lva
  - beats ethantron in low tc, loses in high tc

TODO:
- transposition tables (already implemented position hashing for 3fold repetition)
- optimizations