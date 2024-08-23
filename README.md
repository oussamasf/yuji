# YUji

A **Redis** server implementation from scratch, designed to support basic Redis commands, read RDB files, handle replication, and manage streams.

## Motivation

While mocking a cache system, I realized it would be both challenging and educational to create a Redis server from scratch. This project evolved from that initial idea, allowing me to deepen my understanding of caching, distributed systems, and Redis internals.

## Features

- **Basic Commands**: Supports common Redis commands for key-value operations.
- **RDB File Parsing**: Reads and interprets RDB (Redis Database) files.
- **Handle Replication Handshake**: Establishes and maintains the connection between primary and replica servers, managing the initial synchronization and ongoing updates.
- **Transaction Command Handling**: Supports Redis transaction commands like `MULTI`, `EXEC`, and `DISCARD`, allowing atomic execution of grouped commands.
- **Stream Management**: Manages and processes stream data with blocking read capabilities, with plans to expand stream-related functionality.

## Key Challenges

One of the most challenging components of this server is the implementation of the RESP (Redis Serialization Protocol) parser and the command parser. Unlike many implementations that rely on existing libraries, I intentionally built these components from scratch to gain a deeper understanding of how Redis works internally.

## Installation

```
bash git clone https://github.com/oussamasf/yuji.git
cd yuji
go build -o redis-server`
./redis-server
```

## Usage

Connect to the server using a **_Telnet_** or use a custom client you can create for more advanced interactions. The server is capable of handling basic commands like `SET`, `GET`, and more.
use `./resp.sh` to encode resp commands

## Planned Improvements

- Expanding support for more complex Redis commands.
- Optimizing RDB parsing and error handling (there is a bug in saving binary dump file)
- Enhancing replication logic and stream processing features.

## Contributing

Feel free to contact me if you have ideas for improving the project. Contributions are always welcome.
