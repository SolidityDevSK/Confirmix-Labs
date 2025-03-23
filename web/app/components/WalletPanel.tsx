'use client';

import { useState } from 'react';
import { api } from '../client-api';
import { useBlockchain } from '../contexts/BlockchainContext';

type WalletPanelProps = {
  wallet: { address: string; publicKey: string; privateKey?: string } | null;
  onRefresh: () => void;
};

export default function WalletPanel({ wallet, onRefresh }: WalletPanelProps) {
  // State for wallet actions
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [balance, setBalance] = useState<number | null>(null);
  
  // Transfer form state
  const [transferTo, setTransferTo] = useState('');
  const [transferAmount, setTransferAmount] = useState('');
  const [transferLoading, setTransferLoading] = useState(false);
  const [transferError, setTransferError] = useState<string | null>(null);
  const [transferSuccess, setTransferSuccess] = useState<string | null>(null);
  
  // Import wallet form state
  const [activeTab, setActiveTab] = useState<'create' | 'import'>('create');
  const [privateKey, setPrivateKey] = useState('');
  const [showPrivateKey, setShowPrivateKey] = useState(false);
  
  // Get blockchain context
  const { setWallet, createWallet, importWallet } = useBlockchain();
  
  // Handle wallet creation
  const handleCreateWallet = async () => {
    setLoading(true);
    setError(null);
    setSuccess(null);
    try {
      const newWallet = await createWallet();
      setSuccess(`Wallet created successfully. Make sure to save your private key: ${newWallet.privateKey}`);
      // Save private key to local storage for demo purposes
      localStorage.setItem('walletPrivateKey', newWallet.privateKey);
      // Reset balance when a new wallet is created
      setBalance(null);
      onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred while creating the wallet');
    } finally {
      setLoading(false);
    }
  };
  
  // Handle wallet import
  const handleImportWallet = async () => {
    if (!privateKey.trim()) {
      setError('Private key is required');
      return;
    }
    
    setLoading(true);
    setError(null);
    setSuccess(null);
    try {
      const importedWallet = await importWallet(privateKey.trim());
      setSuccess(`Wallet imported successfully: ${importedWallet.address}`);
      // Reset balance
      setBalance(null);
      // Reset form
      setPrivateKey('');
      onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred while importing the wallet');
    } finally {
      setLoading(false);
    }
  };

  const handleCheckBalance = async () => {
    if (!wallet) return;
    
    setLoading(true);
    setError(null);
    try {
      const balance = await api.getBalance(wallet.address);
      setBalance(balance);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred while checking the balance');
    } finally {
      setLoading(false);
    }
  };

  const handleTransfer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!wallet) return;

    setTransferLoading(true);
    setTransferError(null);
    setTransferSuccess(null);

    try {
      // Convert transfer amount to number
      const amount = parseFloat(transferAmount);
      if (isNaN(amount) || amount <= 0) {
        throw new Error('Invalid transfer amount');
      }

      // Execute the transfer transaction
      const response = await fetch('/api/transaction', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          from: wallet.address,
          to: transferTo,
          value: amount
        }),
      });

      const data = await response.json();

      if (data.error) {
        throw new Error(data.error);
      }

      if (data.warning) {
        // Transaction started but outcome is uncertain
        setTransferSuccess(data.warning);
      } else {
        // Execute the transfer transaction
        setTransferSuccess(data.message || 'Transfer completed successfully');
      }

      // Clear form fields
      setTransferTo('');
      setTransferAmount('');

      // Check balance after 5 seconds
      setTimeout(() => {
        handleCheckBalance();
      }, 5000);

    } catch (err) {
      setTransferError(err instanceof Error ? err.message : 'An error occurred during transfer');
    } finally {
      setTransferLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow">
      <div className="px-4 py-5 sm:p-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-6">Wallet Operations</h2>

        {error && (
          <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-3">
                <p className="text-sm text-red-700">{error}</p>
              </div>
            </div>
          </div>
        )}

        {success && (
          <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-md">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-3">
                <p className="text-sm text-green-700">{success}</p>
              </div>
            </div>
          </div>
        )}

        {!wallet ? (
          <div className="text-center">
            <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
            </svg>
            <h3 className="mt-2 text-sm font-medium text-gray-900">No Wallet Found</h3>
            <p className="mt-1 text-sm text-gray-500">Create a new wallet or import an existing one.</p>
            
            {/* Tabs for wallet creation or import */}
            <div className="mt-4 border-b border-gray-200">
              <div className="-mb-px flex space-x-8" aria-label="Tabs">
                <button
                  onClick={() => setActiveTab('create')}
                  className={`${
                    activeTab === 'create'
                      ? 'border-blue-500 text-blue-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm flex-1`}
                >
                  Create New Wallet
                </button>
                <button
                  onClick={() => setActiveTab('import')}
                  className={`${
                    activeTab === 'import'
                      ? 'border-blue-500 text-blue-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm flex-1`}
                >
                  Import Wallet
                </button>
              </div>
            </div>
            
            {/* Tab content */}
            <div className="mt-6">
              {activeTab === 'create' ? (
                <div>
                  <p className="mb-4 text-sm text-gray-600">
                    Create a new wallet to start interacting with the blockchain
                  </p>
                  <button
                    type="button"
                    onClick={handleCreateWallet}
                    disabled={loading}
                    className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                  >
                    {loading ? (
                      <>
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Creating...
                      </>
                    ) : (
                      'Create New Wallet'
                    )}
                  </button>
                </div>
              ) : (
                <div>
                  <p className="mb-4 text-sm text-gray-600">
                    Import your existing wallet using your private key
                  </p>
                  <div className="mt-1 mb-4">
                    <input
                      type="text"
                      value={privateKey}
                      onChange={(e) => setPrivateKey(e.target.value)}
                      className="shadow-sm focus:ring-blue-500 focus:border-blue-500 block w-full sm:text-sm border-gray-300 rounded-md"
                      placeholder="Enter your private key"
                      required
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handleImportWallet}
                    disabled={loading}
                    className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                  >
                    {loading ? (
                      <>
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Importing...
                      </>
                    ) : (
                      'Import Wallet'
                    )}
                  </button>
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="space-y-6">
            <div className="bg-gray-50 p-4 rounded-lg">
              <h3 className="text-lg font-medium text-gray-900 mb-2">Wallet Information</h3>
              <div className="grid grid-cols-1 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">Address</label>
                  <div className="mt-1 flex rounded-md shadow-sm">
                    <input
                      type="text"
                      readOnly
                      value={wallet.address}
                      className="flex-1 min-w-0 block w-full px-3 py-2 rounded-md border-gray-300 bg-gray-100 text-gray-500 sm:text-sm"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Public Key</label>
                  <div className="mt-1 flex rounded-md shadow-sm">
                    <input
                      type="text"
                      readOnly
                      value={wallet.publicKey}
                      className="flex-1 min-w-0 block w-full px-3 py-2 rounded-md border-gray-300 bg-gray-100 text-gray-500 sm:text-sm"
                    />
                  </div>
                </div>
                {wallet.privateKey && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700 flex items-center justify-between">
                      <span>Private Key</span>
                      <button
                        type="button"
                        onClick={() => setShowPrivateKey(!showPrivateKey)}
                        className="text-xs text-blue-600 hover:text-blue-800 focus:outline-none"
                      >
                        {showPrivateKey ? 'Hide' : 'Show'}
                      </button>
                    </label>
                    <div className="mt-1 flex rounded-md shadow-sm">
                      <input
                        type={showPrivateKey ? "text" : "password"}
                        readOnly
                        value={wallet.privateKey}
                        className="flex-1 min-w-0 block w-full px-3 py-2 rounded-md border-gray-300 bg-gray-100 text-gray-500 sm:text-sm"
                      />
                    </div>
                    <p className="mt-1 text-xs text-red-500">
                      Keep your private key secure! Never share it with anyone.
                    </p>
                  </div>
                )}
              </div>
            </div>

            <div className="bg-gray-50 p-4 rounded-lg">
              <h3 className="text-lg font-medium text-gray-900 mb-2">Balance Information</h3>
              <div className="flex items-center justify-between">
                <div>
                  {balance !== null ? (
                    <p className="text-2xl font-bold text-gray-900">{balance} coin</p>
                  ) : (
                    <p className="text-gray-500">Check to view balance</p>
                  )}
                </div>
                <button
                  type="button"
                  onClick={handleCheckBalance}
                  disabled={loading}
                  className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                >
                  {loading ? (
                    <>
                      <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      Checking...
                    </>
                  ) : (
                    'Check Balance'
                  )}
                </button>
              </div>
            </div>

            <div className="bg-gray-50 p-4 rounded-lg">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Transfer Transaction</h3>
              
              {transferError && (
                <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg className="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <p className="text-sm text-red-700">{transferError}</p>
                    </div>
                  </div>
                </div>
              )}

              {transferSuccess && (
                <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-md">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg className="h-5 w-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <p className="text-sm text-green-700">{transferSuccess}</p>
                    </div>
                  </div>
                </div>
              )}

              <form onSubmit={handleTransfer} className="space-y-4">
                <div>
                  <label htmlFor="transferTo" className="block text-sm font-medium text-gray-700">
                    Recipient Address
                  </label>
                  <div className="mt-1">
                    <input
                      type="text"
                      id="transferTo"
                      value={transferTo}
                      onChange={(e) => setTransferTo(e.target.value)}
                      className="shadow-sm focus:ring-blue-500 focus:border-blue-500 block w-full sm:text-sm border-gray-300 rounded-md"
                      placeholder="0x..."
                      required
                    />
                  </div>
                </div>

                <div>
                  <label htmlFor="transferAmount" className="block text-sm font-medium text-gray-700">
                    Transfer Amount
                  </label>
                  <div className="mt-1">
                    <input
                      type="number"
                      id="transferAmount"
                      value={transferAmount}
                      onChange={(e) => setTransferAmount(e.target.value)}
                      className="shadow-sm focus:ring-blue-500 focus:border-blue-500 block w-full sm:text-sm border-gray-300 rounded-md"
                      placeholder="0.00"
                      step="any"
                      min="0"
                      required
                    />
                  </div>
                </div>

                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={transferLoading}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                  >
                    {transferLoading ? (
                      <>
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Transferring...
                      </>
                    ) : (
                      'Transfer'
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
} 