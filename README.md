# ConfirmixLabs

This project is a hybrid blockchain network that combines Proof of Authority (PoA) and Proof of Humanity (PoH) consensus mechanisms, developed using the Go programming language.

## Project Structure

```
confirmix/
├── main.go            # Main entry point
├── pkg/               # Package directory
│   ├── blockchain/    # Blockchain implementation
│   ├── consensus/     # PoA and PoH consensus mechanisms
│   └── util/          # Utility functions
├── examples/          # Example applications
│   └── web_server.go  # Example web server implementation
├── web/               # Next.js web interface
│   ├── app/           # Web application
│   │   ├── components/# UI components
│   │   └── lib/       # Helper libraries and API services
├── go.mod             # Go module definition
└── README.md          # This file
```

## Current Features

### Blockchain Core
- [x] Basic blockchain data structure
- [x] Block creation and validation
- [x] Transaction management
- [x] Genesis block creation

### Consensus Mechanism
- [x] PoA (Proof of Authority) implementation
- [x] Validator management
- [x] Human verification integration (PoH)
- [x] Hybrid consensus engine
- [x] Block time adjustment (currently 5 seconds)

### API and Web Interface
- [x] RESTful API
- [x] Modern Next.js web interface
- [x] TypeScript support
- [x] Responsive design with Tailwind CSS
- [x] Transaction creation and monitoring
- [x] Blockchain state visualization

## Installation and Running

### Requirements

- Go 1.21 or higher
- Node.js 18 or higher
- npm or yarn

### Starting the Blockchain Server

```bash
# Clone the project
git clone https://github.com/ConfirmixLabs/confirmix.git
cd confirmix

# Start the blockchain server
go run main.go
```

### Starting the Web Interface

```bash
cd web

# Install dependencies
npm install

# Start the development server
npm run dev
```

## Current Status

Currently, the system can perform the following basic functions:
1. Start and manage the blockchain network
2. Create and manage validator nodes
3. Create and validate transactions
4. Create blocks
5. Monitor the blockchain through the web interface

## Future Developments

### 1. Block and Transaction Validation
- [ ] Transaction signature validation
- [ ] Block signature validation
- [ ] Transaction balance verification

### 2. Account/Balance System
- [ ] Account balance tracking
- [ ] Genesis block initial balances
- [ ] Correct balance transfer processing

### 3. Smart Contracts
- [ ] Basic smart contract support
- [ ] Smart contract code execution
- [ ] Smart contract status storage

### 4. Network Layer
- [ ] P2P network support
- [ ] Block and transaction synchronization
- [ ] New node joining the network

### 5. Storage
- [ ] Block permanent storage
- [ ] Status database implementation

## Contributing

This project is in development. To contribute:
1. Fork this repository
2. Create a new branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push your branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## License

MIT 