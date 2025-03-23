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

### 2. Konsensüs Mekanizmaları (`pkg/consensus/`)

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

### 4. Ağ Katmanı (`pkg/network/`)

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

## İşlevsellik ve Özellikler

### İşlem Tipleri

1. **Standart İşlemler**:
   - Adresler arası değer transferi
   - Basit ödeme işlemleri
   - Miktar ve ücret mekanizması

### Blok Üretimi
- 5 saniyelik blok süresi
- İmzalı ve doğrulanabilir bloklar
- Round-robin validatör seçimi
- İşlem bekleyen işlem olduğunda otomatik blok oluşturma

### İşlemler
- Kriptografik olarak imzalı işlemler
- Düzenli değer transferleri
- Akıllı sözleşme işlemleri (dağıtım ve çağrı)
- İşlem havuzu yönetimi

### İnsan Doğrulama
- Validatör olabilmek için insan doğrulaması gereksinimi
- Doğrulama belirteçleri ve zaman aşımı kontrolü
- Harici doğrulama sistemleriyle entegrasyon seçeneği

### Web API ve Arayüz
- RESTful API endpoint'leri
- Blok ve işlem sorgulama
- İşlem gönderme
- Validatör yönetimi ve durum izleme

## Kullanım

Sistem şu anda temel işlevleri gerçekleştirebiliyor:

1. Blockchain ağını başlatma ve yönetme
2. Validatör düğümü oluşturma ve yönetme
3. İşlem oluşturma ve blok zincirine ekleme
4. Web arayüzü üzerinden işlemleri ve blokları görüntüleme
5. İşlem oluşturma ve gönderme
6. Otomatik blok oluşturma (sadece bekleyen işlem varsa)

### Blockchain Sunucusunu Başlatma

```bash
# Standart node başlatma
./blockchain node --address=127.0.0.1 --port=8000

# Validatör node başlatma
./blockchain node --validator=true --poh-verify=true --port=8000
```

### Starting the Blockchain Server

```bash
# Start a standard node
./blockchain node --address=127.0.0.1 --port=8000

# Start a validator node
./blockchain node --validator=true --poh-verify=true --port=8000
```

### Web Sunucusu Örneğini Çalıştırma

```bash
# Örnek web sunucusunu başlatma
go run examples/web_server.go
```

### Running the Web Server Example

```bash
# Start the example web server
go run examples/web_server.go
```

## Mevcut Durum ve Gelecek Geliştirmeler

### Mevcut Özellikler
- [x] Temel blockchain veri yapısı
- [x] Blok oluşturma ve doğrulama
- [x] İşlem (transaction) yönetimi
- [x] Genesis bloğu oluşturma
- [x] Proof of Authority (PoA) implementasyonu
- [x] Validator yönetimi
- [x] İnsan doğrulama entegrasyonu (PoH)
- [x] Hibrit konsensüs motoru
- [x] HTTP API endpoints
- [x] Temel akıllı sözleşme desteği

### Gelecek Geliştirmeler

1. **Blok ve İşlem Doğrulama**
   - [ ] İşlem imzalarının doğrulanmasının iyileştirilmesi
   - [ ] Blok imzalarının doğrulanmasının iyileştirilmesi
   - [ ] İşlem bakiyelerinin kontrolü

2. **Hesap/Bakiye Sistemi**
   - [ ] Hesap bakiyelerinin takibi
   - [ ] Genesis bloğunda başlangıç bakiyeleri
   - [ ] Bakiye transferlerinin doğru işlenmesi

3. **Akıllı Sözleşmeler**
   - [ ] Daha kapsamlı akıllı sözleşme desteği
   - [ ] Sözleşme kodunun yürütülmesinin geliştirilmesi
   - [ ] Sözleşme durumunun kalıcı saklanması

4. **Ağ Katmanı**
   - [ ] P2P ağ desteğinin geliştirilmesi
   - [ ] Blok ve işlem senkronizasyonunun iyileştirilmesi
   - [ ] Yeni düğümlerin ağa katılma sürecinin otomatikleştirilmesi

5. **Depolama**
   - [ ] Blokların kalıcı depolanması
   - [ ] Durum veritabanı implementasyonu
   - [ ] Verimli durum sorgulaması

## Teknik Detaylar

- **Programlama Dili**: Go 1.24
- **Web Sunucusu**: Gorilla Mux
- **Kriptografi**: ECDSA (Elliptic Curve Digital Signature Algorithm)
- **Blok Süresi**: 5 saniye
- **Konsensüs**: Hibrit PoA-PoH

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

## Technical Details

- **Programming Language**: Go 1.24
- **Web Server**: Gorilla Mux
``` 