# PoA-PoH Hybrid Blockchain

Bu proje, Go programlama dili kullanılarak geliştirilmiş, Proof of Authority (PoA) ve Proof of Humanity (PoH) konsensüs mekanizmalarını birleştiren hibrit bir blockchain ağıdır.

## Proje Yapısı

```
.
├── pkg/
│   ├── api/             # HTTP API ve Web sunucusu
│   ├── blockchain/      # Temel blockchain implementasyonu
│   ├── consensus/       # PoA ve PoH konsensüs mekanizmaları
│   └── util/           # Yardımcı fonksiyonlar
├── examples/
│   └── web_server.go   # Örnek web sunucusu implementasyonu
├── web/                # Next.js web arayüzü
│   ├── src/
│   │   ├── app/       # Next.js uygulama dizini
│   │   └── lib/       # Yardımcı kütüphaneler ve API servisleri
├── go.mod             # Go modül tanımı
└── README.md          # Bu dosya
```

## Mevcut Özellikler

### Blockchain Çekirdeği
- [x] Temel blockchain veri yapısı
- [x] Blok oluşturma ve doğrulama
- [x] İşlem (transaction) yönetimi
- [x] Genesis bloğu oluşturma

### Konsensüs Mekanizması
- [x] Proof of Authority (PoA) implementasyonu
- [x] Validator yönetimi
- [x] İnsan doğrulama entegrasyonu (PoH)
- [x] Hibrit konsensüs motoru
- [x] Blok zamanı ayarı (şu anda 5 saniye)

### API ve Web Arayüzü
- [x] HTTP API endpoints
- [x] Modern Next.js web arayüzü
- [x] TypeScript desteği
- [x] Tailwind CSS ile responsive tasarım
- [x] İşlem oluşturma ve izleme
- [x] Blok zinciri durumunu görüntüleme

## Kurulum ve Çalıştırma

### Gereksinimler
- Go 1.21 veya üzeri
- Node.js 18 veya üzeri
- npm veya yarn

### Blockchain Sunucusunu Başlatma
```bash
# Projeyi klonlayın
git clone https://github.com/user/poa-poh-hybrid.git
cd poa-poh-hybrid

# Blockchain sunucusunu başlatın
go run examples/web_server.go
```

### Web Arayüzünü Başlatma
```bash
# Web dizinine gidin
cd web

# Bağımlılıkları yükleyin
npm install

# Geliştirme sunucusunu başlatın
npm run dev
```

## Şu Anki Durum

Şu anda sistem aşağıdaki temel işlevleri gerçekleştirebiliyor:
1. Blockchain ağını başlatma ve yönetme
2. Validator düğümü oluşturma ve yönetme
3. İşlem oluşturma ve blok zincirine ekleme
4. Web arayüzü üzerinden işlemleri ve blokları görüntüleme
5. İşlem oluşturma ve gönderme
6. Otomatik blok oluşturma (sadece bekleyen işlem varsa)

## Gelecek Geliştirmeler

### 1. Blok ve İşlem Doğrulama
- [ ] İşlem imzalarının doğrulanması
- [ ] Blok imzalarının doğrulanması
- [ ] İşlem bakiyelerinin kontrolü

### 2. Hesap/Bakiye Sistemi
- [ ] Hesap bakiyelerinin takibi
- [ ] Genesis bloğunda başlangıç bakiyeleri
- [ ] Bakiye transferlerinin doğru işlenmesi

### 3. Akıllı Sözleşmeler
- [ ] Basit akıllı sözleşme desteği
- [ ] Sözleşme kodunun yürütülmesi
- [ ] Sözleşme durumunun saklanması

### 4. Ağ Katmanı
- [ ] P2P ağ desteği
- [ ] Blok ve işlem senkronizasyonu
- [ ] Yeni düğümlerin ağa katılması

### 5. Depolama
- [ ] Blokların kalıcı depolanması
- [ ] Durum veritabanı implementasyonu

## Katkıda Bulunma

Bu proje geliştirme aşamasındadır. Katkıda bulunmak için:
1. Bu depoyu forklayın
2. Yeni bir branch oluşturun (`git checkout -b feature/amazing-feature`)
3. Değişikliklerinizi commit edin (`git commit -m 'Add some amazing feature'`)
4. Branch'inizi push edin (`git push origin feature/amazing-feature`)
5. Bir Pull Request oluşturun

## Lisans

MIT 