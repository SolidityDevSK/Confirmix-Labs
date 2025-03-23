# Confirmix Labs Blockchain

## Overview

This project is a hybrid blockchain built using the Go programming language that combines Proof of Authority (PoA) and Proof of Humanity (PoH) consensus mechanisms. The system provides a secure and efficient blockchain infrastructure where only authenticated human validators can participate in block production.

## Project Structure

The project is developed with a modular architecture, with the main directories being:

```
./
├── cmd/                # Command line application and entry point
├── pkg/                # Main library code
│   ├── blockchain/     # Core blockchain structures
│   ├── consensus/      # PoA and PoH consensus mechanisms
│   ├── network/        # P2P network functions
│   ├── validator/      # Validator management
│   └── util/           # Helper functions
├── examples/           # Example applications
│   ├── web_server.go   # Example web server implementation
│   ├── basic.go        # Basic blockchain functions
│   ├── contract.go     # Smart contract example
│   └── cli_client.go   # Command line client
│   └── network_simulation.go # P2P network simulation
├── cmd/                # Command line application
│   └── blockchain/     # Blockchain node application
├── web/                # Web interface
│   ├── app/            # Next.js application directory
│   ├── public/         # Static files
└── test/               # Test files
```

## Core Components

### 1. Blockchain Core (`pkg/blockchain/`)

- **`block.go`**: Defines block structure and functions
  - Hash calculation
  - Block validation
  - Signature management
  - Human verification token

- **`blockchain.go`**: Contains chain structure and basic functions
  - Blockchain management
  - Transaction pool
  - Validator registration
  - Genesis block creation

- **`transaction.go`**: Defines transaction structure and validation mechanisms
  - Transaction creation and signing
  - Transaction validation
  - Regular and smart contract transactions

- **`contract.go`**: Provides smart contract support
  - Contract deployment
  - Function calls
  - State management

- **`wallet.go`**: Wallet functions and key management
  - Key pair generation
  - Signing
  - Address calculation

- **`crypto.go`**: Performs cryptographic operations
  - Hash functions
  - Signature algorithms
  - Key management

### 2. Consensus Mechanisms (`pkg/consensus/`)

- **`poa.go`**: Proof of Authority consensus algorithm
  - Authorized validators
  - Round-robin block production
  - Block signing and validation

- **`poh.go`**: Proof of Humanity verification system
  - Human verification record
  - Verification tokens
  - Verification time management

- **`hybrid.go`**: Hybrid consensus engine that combines PoA and PoH
  - Validator selection
  - Block production
  - PoA+PoH validation

- **`poh_external.go`**: Integration with external human verification systems
  - Connection to services like BrightID or Proof of Humanity
  - External verification protocols

### 3. API and Web Server (`pkg/api/`)

- **`server.go`**: HTTP API and web server implementation
  - RESTful API endpoints
  - Transaction sending
  - Block and transaction querying
  - Validator management

### 4. Network Layer (`pkg/network/`)

- P2P network functions and node communication
- Block and transaction distribution
- Node discovery and management

## Hybrid Consensus Mechanism

The most important feature of the system is the combination of two consensus mechanisms:

1. **Proof of Authority (PoA)**: 
   - Block production by authorized validators in round-robin fashion
   - Low transaction cost and high throughput
   - Decentralized but controlled network structure

2. **Proof of Humanity (PoH)**:
   - Human verification system to prevent Sybil attacks
   - Verification that each validator is a real human
   - Periodic verification requirement

This hybrid approach has advantages:
- PoA's fast and efficient block production
- PoH's resistance to Sybil attacks
- More decentralized structure
- Scalability and performance balance

## Functionality and Features

### Transaction Types

1. **Standard Transactions**:
   - Value transfer between addresses
   - Simple payment transactions
   - Amount and fee mechanism

### Block Production
- 5-second block time
- Signed and verifiable blocks
- Round-robin validator selection
- Automatic block creation when there are pending transactions

### Transactions
- Cryptographically signed transactions
- Regular value transfers
- Smart contract transactions (deployment and calls)
- Transaction pool management

### Human Verification
- Human verification requirement to become a validator
- Verification tokens and timeout control
- Integration option with external verification systems

### Web API and Interface
- RESTful API endpoints
- Block and transaction querying
- Transaction submission
- Validator management and status monitoring

## Usage

The system currently can perform the following basic functions:

1. Launching and managing the blockchain network
2. Creating and managing validator nodes
3. Creating transactions and adding them to the blockchain
4. Viewing transactions and blocks through the web interface
5. Creating and sending transactions
6. Automatic block creation (only when there are pending transactions)

### Starting the Blockchain Server

```bash
# Start a standard node
./blockchain node --address=127.0.0.1 --port=8000

# Start a validator node
./blockchain node --validator=true --poh-verify=true --port=8000
```

### Running the Web Server Example

```bash
# Start the example web server
go run examples/web_server.go
```

## Current Status and Future Developments

### Current Features
- [x] Basic blockchain data structure
- [x] Block creation and validation
- [x] Transaction management
- [x] Genesis block creation
- [x] Proof of Authority (PoA) implementation
- [x] Validator management
- [x] Human verification integration (PoH)
- [x] Hybrid consensus engine
- [x] HTTP API endpoints
- [x] Basic smart contract support

### Future Developments

1. **Block and Transaction Validation**
   - [ ] Improvement of transaction signature validation
   - [ ] Improvement of block signature validation
   - [ ] Verification of transaction balances

2. **Account/Balance System**
   - [ ] Tracking of account balances
   - [ ] Initial balances in the genesis block
   - [ ] Proper processing of balance transfers

3. **Smart Contracts**
   - [ ] More comprehensive smart contract support
   - [ ] Improvement of contract code execution
   - [ ] Persistent storage of contract state

4. **Network Layer**
   - [ ] Enhancement of P2P network support
   - [ ] Improvement of block and transaction synchronization
   - [ ] Automation of the process for new nodes to join the network

5. **Storage**
   - [ ] Persistent storage of blocks
   - [ ] State database implementation
   - [ ] Efficient state querying

## Technical Details

- **Programming Language**: Go 1.24
- **Web Server**: Gorilla Mux
- **Cryptography**: ECDSA (Elliptic Curve Digital Signature Algorithm)
- **Block Time**: 5 seconds
- **Consensus**: Hybrid PoA-PoH
``` 