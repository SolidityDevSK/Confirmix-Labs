# PoA-PoH Hibrit Blockchain Projesi

## Genel Bakış

Bu proje, Go programlama dili kullanılarak geliştirilmiş, Proof of Authority (PoA) ve Proof of Humanity (PoH) konsensüs mekanizmalarını birleştiren hibrit bir blockchain ağıdır. Sistem, yalnızca doğrulanmış insan validatörlerin blok üretimine katılabildiği güvenli ve verimli bir blockchain altyapısı sağlar.

## Proje Yapısı

Proje, modüler bir mimari ile geliştirilmiş olup, ana dizinler şunlardır:
```
.
├── pkg/                # Ana kütüphane kodları
│   ├── api/            # HTTP API ve Web sunucusu
│   ├── blockchain/     # Temel blockchain implementasyonu
│   ├── consensus/      # PoA ve PoH konsensüs mekanizmaları
│   ├── network/        # P2P ağ işlevleri
│   ├── validator/      # Validatör yönetimi
│   └── util/           # Yardımcı fonksiyonlar
├── examples/           # Örnek uygulamalar
│   ├── web_server.go   # Örnek web sunucusu implementasyonu
│   ├── basic.go        # Temel blockchain işlevleri
│   ├── contract.go     # Akıllı sözleşme örneği
│   ├── external_poh.go # Harici PoH entegrasyonu
│   └── network_simulation.go # P2P ağ simülasyonu
├── cmd/                # Komut satırı uygulaması
│   └── blockchain/     # Ana blockchain node uygulaması
├── bin/                # Derlenen binary dosyalar
├── go.mod              # Go modül tanımı
├── go.sum              # Go bağımlılık doğrulama
├── Makefile            # Derleme ve çalıştırma komutları
└── README.md           # Proje açıklaması
```

## Temel Bileşenler

### 1. Blockchain Çekirdeği (`pkg/blockchain/`)

- **`block.go`**: Blok yapısını ve işlevlerini tanımlar
  - Hash hesaplama
  - Blok doğrulama
  - İmza yönetimi
  - İnsan doğrulama belirteci

- **`blockchain.go`**: Zincir yapısını ve temel işlevleri içerir
  - Blok zinciri yönetimi
  - İşlem havuzu
  - Validatör kaydı
  - Genesis bloğu oluşturma

- **`transaction.go`**: İşlem yapısını ve doğrulama mekanizmalarını tanımlar
  - İşlem oluşturma ve imzalama
  - İşlem doğrulama
  - Düzenli ve akıllı sözleşme işlemleri

- **`contract.go`**: Akıllı sözleşme desteği sağlar
  - Sözleşme dağıtımı
  - Fonksiyon çağrıları
  - Durum yönetimi

- **`wallet.go`**: Cüzdan işlevleri ve anahtar yönetimi
  - Anahtar çifti oluşturma
  - İmzalama
  - Adres hesaplama

- **`crypto.go`**: Kriptografik işlemleri gerçekleştirir
  - Hash fonksiyonları
  - İmza algoritmaları
  - Anahtar yönetimi

### 2. Konsensüs Mekanizmaları (`pkg/consensus/`)

- **`poa.go`**: Proof of Authority konsensüs algoritması
  - Yetkilendirilmiş validatörler
  - Round-robin blok üretimi
  - Blok imzalama ve doğrulama

- **`poh.go`**: Proof of Humanity doğrulama sistemi
  - İnsan doğrulama kaydı
  - Doğrulama belirteçleri
  - Doğrulama süresi yönetimi

- **`hybrid.go`**: PoA ve PoH'yi birleştiren hibrit konsensüs motoru
  - Validatör seçimi
  - Blok üretimi
  - PoA+PoH doğrulaması

- **`poh_external.go`**: Harici insan doğrulama sistemleriyle entegrasyon
  - BrightID veya Proof of Humanity gibi servislere bağlantı
  - Harici doğrulama protokolleri

### 3. API ve Web Sunucusu (`pkg/api/`)

- **`server.go`**: HTTP API ve web sunucusu implementasyonu
  - RESTful API endpoint'leri
  - İşlem gönderme
  - Blok ve işlem sorgulama
  - Validatör yönetimi

### 4. Ağ Katmanı (`pkg/network/`)

- P2P ağ işlevleri ve düğüm iletişimi
- Blok ve işlem dağıtımı
- Düğüm keşfi ve yönetimi

## Hibrit Konsensüs Mekanizması

Sistemin en önemli özelliği, iki konsensüs mekanizmasını birleştirmesidir:

1. **Proof of Authority (PoA)**: 
   - Yetkilendirilmiş validatörler tarafından sırayla blok üretimi
   - Düşük işlem maliyeti ve yüksek verimlilik
   - Merkezi olmayan ancak kontrollü ağ yapısı

2. **Proof of Humanity (PoH)**:
   - İnsan doğrulama sistemi ile Sybil saldırılarını önleme
   - Her validatörün gerçek bir insan olduğunun doğrulanması
   - Periyodik yeniden doğrulama gereksinimi

Bu hibrit yaklaşımın avantajları:
- PoA'nın hızlı ve verimli blok üretimi
- PoH'nin Sybil saldırılarına karşı direnci
- Daha ademi merkeziyetçi bir yapı
- Ölçeklenebilirlik ve performans dengesi

## İşlevsellik ve Özellikler

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

### Web Sunucusu Örneğini Çalıştırma

```bash
# Örnek web sunucusunu başlatma
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
``` 