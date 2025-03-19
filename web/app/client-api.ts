import { BlockchainStatus, Block, Transaction, ValidatorInfo } from './lib/types';

export const api = {
  getStatus: async (): Promise<BlockchainStatus> => {
    const response = await fetch('/api/status');
    if (!response.ok) {
      throw new Error('API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getBlocks: async (): Promise<Block[]> => {
    const response = await fetch('/api/blocks');
    if (!response.ok) {
      throw new Error('API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getTransactions: async (): Promise<Transaction[]> => {
    const response = await fetch('/api/transactions');
    if (!response.ok) {
      throw new Error('API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getPendingTransactions: async (): Promise<Transaction[]> => {
    const response = await fetch('/api/transactions/pending');
    if (!response.ok) {
      throw new Error('API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getConfirmedTransactions: async (): Promise<Transaction[]> => {
    const response = await fetch('/api/transactions/confirmed');
    if (!response.ok) {
      throw new Error('API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getValidators: async (): Promise<ValidatorInfo[]> => {
    const response = await fetch('/api/validators');
    if (!response.ok) {
      throw new Error('API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  registerValidator: async (address: string, humanProof: string): Promise<ValidatorInfo> => {
    const response = await fetch('/api/validator/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address, humanProof })
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  createWallet: async (): Promise<{ address: string; publicKey: string }> => {
    const response = await fetch('/api/wallet/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'API bağlantı hatası');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  async getBalance(address: string): Promise<number> {
    try {
      const response = await fetch(`/api/wallet/balance/${address}`);
      if (!response.ok) {
        throw new Error('Bakiye bilgisi alınamadı');
      }
      const data = await response.json();
      return data.balance;
    } catch (error) {
      console.error('Bakiye sorgulama hatası:', error);
      throw new Error('Bakiye bilgisi alınamadı');
    }
  },

  transfer: async (from: string, to: string, value: number): Promise<any> => {
    const response = await fetch('/api/transaction', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ from, to, value })
    });
    
    const data = await response.json();
    if (!response.ok || data.error) {
      throw new Error(data.error || 'Transfer işlemi başarısız oldu');
    }
    return data;
  }
}; 