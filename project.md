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
- Option for integration with external verification systems

### Web API and Interface
- RESTful API endpoints
- Block and transaction querying
- Transaction sending
- Validator management and status monitoring

## Usage

The system currently can perform basic functions:

1. Starting and managing the blockchain network
2. Creating and managing validator nodes
3. Creating transactions and adding them to the blockchain
4. Viewing transactions and blocks through the web interface
5. Creating and sending transactions
6. Automatic block creation (only if there are pending transactions)

### Starting the Blockchain Server

```bash
# Start a standard node
./blockchain node --address=127.0.0.1 --port=8000

# Start a validator node
./blockchain node --validator=true --poh-verify=true --port=8000

# Start a node with an initial admin
./blockchain node --validator-mode=admin --admin=<YOUR_WALLET_ADDRESS> --port=8000
```

### Running the Web Server Example

```bash
# Start the example web server
go run examples/web_server.go
```

## Admin and Validator Management

The system implements a secure model for managing administrators and validators in a hybrid approach that can evolve from centralized to decentralized governance.

### Setting Up the First Admin

When initializing the blockchain for the first time, you can specify the first admin address:

```bash
./blockchain node --validator-mode=admin --admin=<YOUR_WALLET_ADDRESS> --port=8000
```

* The `--validator-mode=admin` parameter specifies that the validator approval will be managed by administrators.
* The `--admin=<YOUR_WALLET_ADDRESS>` parameter sets the initial admin address.
* This first admin will only be initialized if there are no existing admins in the system.

### Admin Management API

The following API endpoints are available for admin management:

1. **List All Admins**
   ```
   GET /api/admin/list
   ```
   Returns a list of all current admin addresses.

2. **Add a New Admin**
   ```
   POST /api/admin/add
   ```
   
   Request body (must be signed by an existing admin):
   ```json
   {
     "action": "addAdmin",
     "data": {
       "address": "<NEW_ADMIN_ADDRESS>"
     },
     "adminAddress": "<CURRENT_ADMIN_ADDRESS>",
     "signature": "<SIGNATURE>",
     "timestamp": <UNIX_TIMESTAMP>
   }
   ```

3. **Remove an Admin**
   ```
   POST /api/admin/remove
   ```
   
   Request body (must be signed by an existing admin):
   ```json
   {
     "action": "removeAdmin",
     "data": {
       "address": "<ADMIN_ADDRESS_TO_REMOVE>"
     },
     "adminAddress": "<CURRENT_ADMIN_ADDRESS>",
     "signature": "<SIGNATURE>",
     "timestamp": <UNIX_TIMESTAMP>
   }
   ```
   
   Note: The system prevents removing the last admin to ensure there's always at least one admin.

### Validator Management API

The following API endpoints are available for validator management:

1. **Register as a Validator**
   ```
   POST /api/validators/register
   ```
   
   Request body:
   ```json
   {
     "address": "<WALLET_ADDRESS>",
     "humanProof": "<HUMAN_PROOF_TOKEN>"
   }
   ```
   
   This registers the address as a potential validator. Depending on the validator mode, it may be automatically approved or require admin approval.

2. **Approve a Validator**
   ```
   POST /api/admin/validators/approve
   ```
   
   Request body (must be signed by an admin):
   ```json
   {
     "action": "approveValidator",
     "data": {
       "address": "<VALIDATOR_ADDRESS>"
     },
     "adminAddress": "<ADMIN_ADDRESS>",
     "signature": "<SIGNATURE>",
     "timestamp": <UNIX_TIMESTAMP>
   }
   ```

3. **Reject a Validator**
   ```
   POST /api/admin/validators/reject
   ```
   
   Request body (must be signed by an admin):
   ```json
   {
     "action": "rejectValidator",
     "data": {
       "address": "<VALIDATOR_ADDRESS>",
       "reason": "<REJECTION_REASON>"
     },
     "adminAddress": "<ADMIN_ADDRESS>",
     "signature": "<SIGNATURE>",
     "timestamp": <UNIX_TIMESTAMP>
   }
   ```

4. **Suspend a Validator**
   ```
   POST /api/admin/validators/suspend
   ```
   
   Request body (must be signed by an admin):
   ```json
   {
     "action": "suspendValidator",
     "data": {
       "address": "<VALIDATOR_ADDRESS>",
       "reason": "<SUSPENSION_REASON>"
     },
     "adminAddress": "<ADMIN_ADDRESS>",
     "signature": "<SIGNATURE>",
     "timestamp": <UNIX_TIMESTAMP>
   }
   ```

5. **List All Validators**
   ```
   GET /api/validators
   ```
   
   Returns a list of all validators, including their status and information.

### Security Features

All admin API operations require:

1. **Request Signing**: Each request must be signed by the admin's private key to prove authenticity.
2. **Timestamp Validation**: Requests include a timestamp to prevent replay attacks. Requests older than 5 minutes are rejected.
3. **Admin Verification**: Only registered admins can perform admin operations.
4. **Last Admin Protection**: The system prevents removing the last admin to ensure administrative access is maintained.

### Validator Mode Options

The system supports multiple validator approval modes:

1. **Admin Only** (`--validator-mode=admin`): Only administrators can approve validators.
2. **Hybrid** (`--validator-mode=hybrid`): Both administrators and governance votes can approve validators.
3. **Governance** (`--validator-mode=governance`): Only governance votes can approve validators (requires enabling governance).
4. **Automatic** (`--validator-mode=automatic`): Validators are automatically approved if they meet criteria.

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
   - [ ] Improving transaction signature verification
   - [ ] Improving block signature verification
   - [ ] Checking transaction balances

2. **Account/Balance System**
   - [ ] Tracking account balances
   - [ ] Initial balances in genesis block
   - [ ] Proper processing of balance transfers

3. **Smart Contracts**
   - [ ] More comprehensive smart contract support
   - [ ] Improving contract code execution
   - [ ] Persistent storage of contract state

4. **Network Layer**
   - [ ] Improving P2P network support
   - [ ] Improving block and transaction synchronization
   - [ ] Automating the process of new nodes joining the network

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
``` 