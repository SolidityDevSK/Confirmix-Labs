'use client';

import { useEffect, useState } from 'react';
import { api, BlockchainStatus, Block, Transaction, testWallet } from './lib/api';

type TabType = 'overview' | 'blocks' | 'transactions';

export default function Home() {
    const [status, setStatus] = useState<BlockchainStatus | null>(null);
    const [blocks, setBlocks] = useState<Block[]>([]);
    const [transactions, setTransactions] = useState<Transaction[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [activeTab, setActiveTab] = useState<TabType>('overview');

    // Form state
    const [txForm, setTxForm] = useState({
        from: testWallet.address,
        to: '',
        value: '',
        data: '',
    });
    const [showNewTxForm, setShowNewTxForm] = useState(false);

    // Fetch data
    const fetchData = async () => {
        try {
            setLoading(true);
            setError(null);
            
            const [statusData, blocksData, txData] = await Promise.all([
                api.getStatus(),
                api.getBlocks(),
                api.getTransactions(),
            ]);

            setStatus(statusData);
            setBlocks(blocksData);
            setTransactions(txData);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Bir hata oluştu');
        } finally {
            setLoading(false);
        }
    };

    // Create transaction
    const handleCreateTransaction = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            setError(null);
            await api.createTransaction({
                from: txForm.from,
                to: txForm.to,
                value: parseFloat(txForm.value),
                data: txForm.data || undefined
            });
            setTxForm({ from: testWallet.address, to: '', value: '', data: '' });
            setShowNewTxForm(false);
            fetchData();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'İşlem oluşturulamadı');
        }
    };

    // Initial load and auto-refresh
    useEffect(() => {
        fetchData();
        const interval = setInterval(fetchData, 10000);
        return () => clearInterval(interval);
    }, []);

    if (loading && !status) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            </div>
        );
    }

    return (
        <div className="min-h-screen">
            {/* Header */}
            <header className="bg-white shadow-sm">
                <div className="container mx-auto px-4 py-4">
                    <div className="flex justify-between items-center">
                        <h1 className="text-xl font-bold text-gray-800">Blockchain Explorer</h1>
                        {status && (
                            <div className="flex items-center space-x-4 text-sm text-gray-600">
                                <div>Yükseklik: {status.height}</div>
                                <div className="hidden md:block">Son Blok: {status.lastBlock.substring(0, 8)}...</div>
                            </div>
                        )}
                    </div>
                </div>
            </header>

            {/* Error Display */}
            {error && (
                <div className="fixed top-4 right-4 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded shadow-lg">
                    <button onClick={() => setError(null)} className="float-right font-bold">&times;</button>
                    {error}
                </div>
            )}

            {/* Main Content */}
            <main className="container mx-auto px-4 py-6">
                {/* Tabs */}
                <div className="flex border-b mb-6">
                    <button
                        onClick={() => setActiveTab('overview')}
                        className={`px-4 py-2 -mb-px ${activeTab === 'overview' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-gray-600'}`}
                    >
                        Genel Bakış
                    </button>
                    <button
                        onClick={() => setActiveTab('blocks')}
                        className={`px-4 py-2 -mb-px ${activeTab === 'blocks' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-gray-600'}`}
                    >
                        Bloklar
                    </button>
                    <button
                        onClick={() => setActiveTab('transactions')}
                        className={`px-4 py-2 -mb-px ${activeTab === 'transactions' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-gray-600'}`}
                    >
                        İşlemler
                    </button>
                </div>

                {/* Tab Content */}
                <div className="bg-white rounded-lg shadow">
                    {/* Overview Tab */}
                    {activeTab === 'overview' && (
                        <div className="p-6">
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                                <div className="bg-gray-50 p-4 rounded-lg">
                                    <h3 className="text-lg font-semibold mb-2">Blockchain Durumu</h3>
                                    <div className="text-gray-600">
                                        <p>Yükseklik: {status?.height}</p>
                                        <p className="truncate">Son Blok: {status?.lastBlock}</p>
                                    </div>
                                </div>
                                <div className="bg-gray-50 p-4 rounded-lg">
                                    <h3 className="text-lg font-semibold mb-2">Son İşlemler</h3>
                                    <p className="text-gray-600">Bekleyen: {transactions.length}</p>
                                </div>
                                <div className="bg-gray-50 p-4 rounded-lg">
                                    <h3 className="text-lg font-semibold mb-2">Blok İstatistikleri</h3>
                                    <p className="text-gray-600">Toplam Blok: {blocks.length}</p>
                                </div>
                            </div>

                            <div className="mt-8">
                                <button
                                    onClick={() => setShowNewTxForm(!showNewTxForm)}
                                    className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 transition-colors"
                                >
                                    {showNewTxForm ? 'İşlem Formunu Gizle' : 'Yeni İşlem Oluştur'}
                                </button>

                                {showNewTxForm && (
                                    <form onSubmit={handleCreateTransaction} className="mt-4 bg-gray-50 p-6 rounded-lg">
                                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                            <div>
                                                <label className="block text-sm font-medium text-gray-700 mb-1">Gönderen</label>
                                                <input
                                                    type="text"
                                                    value={txForm.from}
                                                    onChange={e => setTxForm(prev => ({ ...prev, from: e.target.value }))}
                                                    className="w-full p-2 border rounded focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                                                    required
                                                />
                                            </div>
                                            <div>
                                                <label className="block text-sm font-medium text-gray-700 mb-1">Alıcı</label>
                                                <input
                                                    type="text"
                                                    value={txForm.to}
                                                    onChange={e => setTxForm(prev => ({ ...prev, to: e.target.value }))}
                                                    className="w-full p-2 border rounded focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                                                    required
                                                />
                                            </div>
                                        </div>
                                        <div className="mt-4">
                                            <label className="block text-sm font-medium text-gray-700 mb-1">Miktar</label>
                                            <input
                                                type="number"
                                                value={txForm.value}
                                                onChange={e => setTxForm(prev => ({ ...prev, value: e.target.value }))}
                                                className="w-full p-2 border rounded focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                                                required
                                                min="0"
                                                step="0.01"
                                            />
                                        </div>
                                        <div className="mt-4">
                                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                                Veri (İsteğe bağlı, JSON)
                                            </label>
                                            <textarea
                                                value={txForm.data}
                                                onChange={e => setTxForm(prev => ({ ...prev, data: e.target.value }))}
                                                className="w-full p-2 border rounded focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                                                rows={3}
                                            />
                                        </div>
                                        <button
                                            type="submit"
                                            className="mt-4 bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 transition-colors"
                                        >
                                            İşlem Oluştur
                                        </button>
                                    </form>
                                )}
                            </div>
                        </div>
                    )}

                    {/* Blocks Tab */}
                    {activeTab === 'blocks' && (
                        <div className="divide-y">
                            {blocks.map(block => (
                                <div key={block.Hash} className="p-4 hover:bg-gray-50 transition-colors">
                                    <div className="flex justify-between items-start">
                                        <div>
                                            <h3 className="text-lg font-semibold">Blok #{block.Index}</h3>
                                            <p className="text-sm text-gray-500">
                                                {new Date(block.Timestamp * 1000).toLocaleString()}
                                            </p>
                                        </div>
                                        <div className="text-right text-sm text-gray-600">
                                            <p>İşlem Sayısı: {block.Transactions.length}</p>
                                            <p>Validator: {block.Validator.substring(0, 8)}...</p>
                                        </div>
                                    </div>
                                    <div className="mt-2 text-sm text-gray-600">
                                        <p className="font-mono">Hash: {block.Hash}</p>
                                        <p className="font-mono">Önceki Hash: {block.PrevHash}</p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}

                    {/* Transactions Tab */}
                    {activeTab === 'transactions' && (
                        <div className="divide-y">
                            {transactions.map(tx => (
                                <div key={tx.ID} className="p-4 hover:bg-gray-50 transition-colors">
                                    <div className="flex justify-between items-start">
                                        <div>
                                            <h3 className="font-mono text-sm">{tx.ID}</h3>
                                            <p className="text-sm text-gray-500">
                                                {new Date(tx.Timestamp * 1000).toLocaleString()}
                                            </p>
                                        </div>
                                        <div className="text-right">
                                            <p className="text-lg font-semibold">{tx.Value} token</p>
                                        </div>
                                    </div>
                                    <div className="mt-2 text-sm">
                                        <p>
                                            <span className="text-gray-600">Gönderen:</span>{' '}
                                            <span className="font-mono">{tx.From}</span>
                                        </p>
                                        <p>
                                            <span className="text-gray-600">Alıcı:</span>{' '}
                                            <span className="font-mono">{tx.To}</span>
                                        </p>
                                        {tx.Data && (
                                            <p className="mt-2">
                                                <span className="text-gray-600">Veri:</span>{' '}
                                                <span className="font-mono text-xs">{JSON.stringify(tx.Data)}</span>
                                            </p>
                                        )}
                                    </div>
                                </div>
                            ))}
                            {transactions.length === 0 && (
                                <div className="p-8 text-center text-gray-500">
                                    Bekleyen işlem bulunmuyor
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </main>
        </div>
    );
}
