'use client';

import { useEffect, useState } from 'react';
import { api } from './client-api';
import { BlockchainStatus, Block, Transaction } from './lib/types';
import ErrorMessage from './components/ErrorMessage';
import FallbackPage from './components/FallbackPage';
import WalletPanel from './components/WalletPanel';
import ValidatorPanel from './components/ValidatorPanel';
import BlocksTable from './components/BlocksTable';
import TransactionsTable from './components/TransactionsTable';
import Header from './components/Header';

// Tipler için yardımcı fonksiyonlar
const isString = (value: any): value is string => typeof value === 'string';
const isBlockData = (value: any): boolean => value && typeof value === 'object' && 'Hash' in value;

// Son blok bilgisini güvenli şekilde ayrıştıran fonksiyon
const getLastBlockInfo = (blockData: Block | string | any): { Hash: string } => {
  if (isString(blockData)) {
    try {
      // JSON formatındaysa ayrıştır
      return JSON.parse(blockData);
    } catch (e) {
      // JSON formatında değilse string olarak kullan
      return { Hash: blockData };
    }
  } else if (isBlockData(blockData)) {
    // Zaten bir nesne ise ve hash içeriyorsa
    return blockData;
  }
  // Hiçbiri değilse, varsayılan değer döndür
  return { Hash: "Bilinmiyor" };
};

export default function Home() {
  const [status, setStatus] = useState<BlockchainStatus | null>(null);
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [pendingTransactions, setPendingTransactions] = useState<Transaction[]>([]);
  const [validators, setValidators] = useState<Array<{ address: string; humanProof: string }>>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [connectionError, setConnectionError] = useState(false);
  const [usingMockData, setUsingMockData] = useState(false);
  const [retryCount, setRetryCount] = useState(0);
  const [wallet, setWallet] = useState<{ address: string; publicKey: string } | null>(null);
  
  // Aktif sayfa/sekme
  const [activePanel, setActivePanel] = useState<'overview' | 'wallet' | 'validator' | 'blocks' | 'transactions'>('overview');

  // Fetch data
  const fetchData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      // Status, bloklar, işlemler ve validatörleri çek
      const [status, blocks, transactions, pendingTransactions, validators] = await Promise.all([
        api.getStatus(),
        api.getBlocks(),
        api.getTransactions(),
        api.getPendingTransactions(),
        api.getValidators()
      ]);

      setStatus(status);
      setBlocks(blocks);
      setTransactions(transactions);
      setPendingTransactions(pendingTransactions);
      setValidators(validators);
      setConnectionError(false);
      setUsingMockData(false);
      
    } catch (error) {
      console.error('Veri yükleme hatası:', error);
      setError('Backend sunucusuna bağlantı sağlanamadı. Lütfen blockchain sunucusunun çalıştığından emin olun.');
      setConnectionError(true);
      setUsingMockData(false);
    } finally {
      setLoading(false);
    }
  };

  // Polling süresini 5 saniyeye düşür
  useEffect(() => {
    let isSubscribed = true;

    const fetchDataIfSubscribed = async () => {
      if (isSubscribed) {
        await fetchData();
      }
    };

    fetchDataIfSubscribed();
    const interval = setInterval(fetchDataIfSubscribed, 5000);

    return () => {
      isSubscribed = false;
      clearInterval(interval);
    };
  }, [retryCount]);

  // API sunucusuna bağlanılamıyorsa FallbackPage göster
  if (connectionError) {
    return <FallbackPage />;
  }

  // Cüzdan durumunu WalletPanel'e aktaracak şekilde güncelle
  const handleWalletChange = (newWallet: { address: string; publicKey: string } | null) => {
    setWallet(newWallet);
  };

  const handleRefresh = () => {
    fetchData();
  };

  if (loading && !status) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p>Blockchain verileri yükleniyor...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <ErrorMessage 
        message={error} 
        action={() => window.location.reload()} 
        actionText="Sayfayı Yenile" 
      />
    );
  }

  if (!status) {
    return (
      <ErrorMessage 
        message="Blockchain verilerine erişilemiyor. Lütfen daha sonra tekrar deneyin." 
        action={() => window.location.reload()} 
        actionText="Tekrar Dene" 
      />
    );
  }

  return (
    <main className="container mx-auto px-4 py-8">
      {usingMockData && (
        <div className="bg-yellow-100 text-yellow-800 px-4 py-2 text-sm text-center mb-4 rounded">
          API sunucusuna erişimde sorun yaşanıyor. Şu anda gösterilen veriler güncel olmayabilir.
          <button 
            onClick={() => window.location.reload()} 
            className="ml-2 underline hover:no-underline"
          >
            Yenile
          </button>
        </div>
      )}
      
      {/* Header Component */}
      <Header 
        status={status}
        transactionCount={pendingTransactions.length}
        validatorCount={validators.length}
        usingMockData={usingMockData}
        onRefresh={handleRefresh}
      />
      
      {/* Ana Menü */}
      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <div className="flex flex-wrap gap-2 justify-center">
          <button
            onClick={() => setActivePanel('overview')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'overview'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
            </svg>
            Genel Bakış
          </button>
          <button
            onClick={() => setActivePanel('wallet')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'wallet'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
            </svg>
            Cüzdan İşlemleri
          </button>
          <button
            onClick={() => setActivePanel('validator')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'validator'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            Validator İşlemleri
          </button>
          <button
            onClick={() => setActivePanel('blocks')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'blocks'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            Bloklar
          </button>
          <button
            onClick={() => setActivePanel('transactions')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'transactions'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
            </svg>
            İşlemler
          </button>
        </div>
      </div>
      
      {/* Aktif Panel İçeriği */}
      {activePanel === 'overview' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-bold mb-4">Son Bloklar</h2>
            <BlocksTable blocks={blocks.slice(0, 5)} loading={loading} />
            <div className="mt-4 text-right">
              <button 
                onClick={() => setActivePanel('blocks')} 
                className="text-blue-600 hover:text-blue-800"
              >
                Tüm blokları görüntüle →
              </button>
            </div>
          </div>
          
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-bold mb-4">Son İşlemler</h2>
            <TransactionsTable transactions={transactions.slice(0, 5)} loading={loading} />
            <div className="mt-4 text-right">
              <button 
                onClick={() => setActivePanel('transactions')} 
                className="text-blue-600 hover:text-blue-800"
              >
                Tüm işlemleri görüntüle →
              </button>
            </div>
          </div>
        </div>
      )}
      
      {activePanel === 'wallet' && (
        <WalletPanel 
          wallet={wallet}
          onWalletChange={handleWalletChange}
        />
      )}
      
      {activePanel === 'validator' && (
        <ValidatorPanel 
          wallet={wallet}
          validators={validators}
          onRefresh={handleRefresh}
        />
      )}
      
      {activePanel === 'blocks' && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-bold mb-4">Tüm Bloklar</h2>
          <BlocksTable blocks={blocks} loading={loading} />
        </div>
      )}
      
      {activePanel === 'transactions' && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-bold mb-4">Tüm İşlemler</h2>
          <TransactionsTable transactions={transactions} loading={loading} />
        </div>
      )}
    </main>
  );
}
