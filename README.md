# Lacrima
## A chess engine so bad it'll make you cry.
### ALL NON-FULL VERSIONS (1.0.1-1.0.9 etc) ARE DEV VERSIONS AND NOT TO BE TESTED
### ONLY v1.1,v1.2 etc are full releases


written in Go, and hopefully structured better than ethantron (my old Python engine).

DONE:
  - movegen, board rep using bitboards
  - normal moves, captures, promotions, en passant, and castling
  - make/unmake, null moves, attack detection
  - FEN parsing
  - UCI support because talking to GUIs is apparently required
  - position/go/stop/quit/setoption/ucinewgame/isready/uci
  - perft testing for correctness
  - time management and search cancellation
  - texel-tuned eval using PESTO tapered eval with material + piece-square tables as a starting point
  - negamax with a/b pruning
  - iterative deepening
  - aspiration windows
  - transposition tables with exact/lower/upper bounds
  - configurable hash size + clear hash
  - ordered quiescence search
  - null-move pruning
  - repetition detection
  - mate score handling
  - move ordering with TT move, MVV/LVA, promos, killers, and history
  - principal variation search
  - late move reduction
  - reverse futility pruning
  - futility pruning

## special thanks to:
- members of EP and SF discord (cie, matt, seba, jeremy(funny), nano, andrew, kama, lily, chef, dr. extension, jw)

GOAL: 3000 ELO
