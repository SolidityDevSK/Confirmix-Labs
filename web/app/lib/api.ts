// Types
export interface BlockchainStatus {
    height: number;
    lastBlock: string;
}

export interface Block {
    Index: number;
    Hash: string;
    PrevHash: string;
    Validator: string;
    Timestamp: number;
    Transactions: Transaction[];
}

export interface Transaction {
    ID: string;
    From: string;
    To: string;
    Value: number;
    Data?: string;
    Timestamp: number;
    Signature?: string;
}

// Test wallet for development
export const testWallet = {
    address: "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf",
    privateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // Removed 0x prefix
};

// API Base URL
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

// API Service
export const api = {
    // Fetch blockchain status
    async getStatus(): Promise<BlockchainStatus> {
        const response = await fetch(`${API_BASE_URL}/status`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    },

    // Fetch blocks
    async getBlocks(): Promise<Block[]> {
        const response = await fetch(`${API_BASE_URL}/blocks`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    },

    // Fetch pending transactions
    async getTransactions(): Promise<Transaction[]> {
        const response = await fetch(`${API_BASE_URL}/transactions`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    },

    // Create new transaction
    async createTransaction(tx: { from: string; to: string; value: number; data?: string }): Promise<Transaction> {
        // Add timestamp and signature
        const transaction = {
            ...tx,
            Timestamp: Math.floor(Date.now() / 1000),
            // Convert private key to base64
            Signature: btoa(testWallet.privateKey),
            Data: tx.data || undefined
        };

        const response = await fetch(`${API_BASE_URL}/transaction`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(transaction),
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
        }
        
        return response.json();
    },
}; 