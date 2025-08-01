# Akasha Showdown Engine

This service is a standalone real-time game server implemented in Go with a C++ tic-tac-toe game engine.

## Features

- WebSocket server for real-time multiplayer games
- Turn-based Tic-Tac-Toe game logic in C++ via cgo
- In-memory match state for fast turn resolution
- Communicates with external REST API to report game results

## Requirements

- Go 1.16+
- C++ compiler (with C++11 support)

## Building

```bash
go mod tidy
go build -o bin/server cmd/server/main.go
```

## Running

```bash
./bin/server -addr :8080 -api-url http://localhost:8000
```

Clients can connect via WebSocket at `ws://localhost:8080/ws` and send/receive JSON messages.

## Playing the Game

After connecting to the WebSocket endpoint, two clients are paired automatically:

- The first client to connect will receive:

  ```json
  { "type": "waiting_for_opponent" }
  ```

  indicating it's player 1 and is waiting for an opponent.

- When a second client connects, both clients will receive a `start` message with the initial board state and the current player:

  ```json
  {
    "type": "start",
    "payload": {
      "board": [
        [0, 0, 0],
        [0, 0, 0],
        [0, 0, 0]
      ],
      "current_player": 1,
      "winner": 0
    }
  }
  ```

Players then take turns sending `play_move` messages:

```json
{"type":"play_move","payload":{"row":<0-2>,"col":<0-2>}}
```

Rows and columns are zero-indexed (0–2). Each valid move triggers a `state_update` broadcast:

```json
{
  "type": "state_update",
  "payload": {
    "board": [
      [1, 0, 0],
      [0, 2, 0],
      [0, 0, 0]
    ],
    "current_player": 1,
    "winner": 0
  }
}
```

When the `winner` field becomes nonzero (1 or 2), the game ends and the connections are closed.

## Docker

### Prerequisites

- Docker 19.03+

### Build Docker image

```bash
docker build -t akasha-showdown-engine .
```

### Run container

```bash
docker run -p 8080:8080 akasha-showdown-engine
```

To customize the REST API endpoint:

```bash
docker run -p 8080:8080 akasha-showdown-engine \
  -api-url http://api-server:8000
```

## Makefile

You can also use the provided Makefile to build and run the Docker container:

```bash
make docker-build    # build the Docker image
make docker-run      # run the server container
make docker-shell    # open a shell inside the container
```
