// Type definitions only

export interface BlockchainStatus {
    height: number;
    lastBlock: string;
}

export interface Block {
    index: number;
    timestamp: number;
    transactions: Transaction[];
    prevHash: string;
    hash: string;
    validator: string;
}

export interface Transaction {
    id: string;
    from: string;
    to: string;
    value: number;
    data?: string;
    timestamp: number;
    signature: string;
}

export interface ValidatorInfo {
    address: string;
    humanProof: string;
}

export interface TransactionResult extends Transaction {
    blockMined?: boolean; // Flag to indicate if transaction was immediately mined
    status?: string;
    message?: string;
}

export interface Wallet {
    address: string;
    publicKey: string;
}

// API interface definition
export interface Api {
    getStatus(): Promise<BlockchainStatus>;
    getBlocks(): Promise<Block[]>;
    getTransactions(): Promise<Transaction[]>;
    createWallet(): Promise<{ address: string; publicKey: string }>;
    getBalance(address: string): Promise<number>;
    createTransaction(from: string, to: string, amount: number, data?: string): Promise<TransactionResult>;
    mineBlock(validator: string): Promise<Block>;
    registerValidator(data: { address: string; humanProof: string }): Promise<{ message: string; address: string }>;
    getValidators(): Promise<ValidatorInfo[]>;
} 