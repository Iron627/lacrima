# Lacrima
## A chess engine so bad it'll make you cry.

written in Go, and hopefully structured better than ethantron (my old Python engine).

DONE:
  - movegen, board rep, and legality filtering
  - normal moves, captures, promotions, en passant, and castling
  - make/unmake, null moves, attack detection, and king-square caching
  - FEN parsing
  - UCI support because talking to GUIs is apparently required
  - position/go/stop/quit/setoption/ucinewgame/isready/uci
  - custom perft command
  - time management and search cancellation
  - PESTO tapered eval with material + piece-square tables
  - negamax with a/b pruning
  - iterative deepening
  - aspiration windows
  - transposition tables with exact/lower/upper bounds
  - configurable hash size + clear hash
  - ordered quiescence search
  - null-move pruning
  - 1-ply check extensions
  - repetition detection
  - mate score handling
  - move ordering with TT move, MVV/LVA-ish captures, promos, killers, and history
  - principal variation search
  - late move reduction
  - reverse futility pruning

GOAL: 2000 ELO

current estimate vs stash v13: 1954 elo
