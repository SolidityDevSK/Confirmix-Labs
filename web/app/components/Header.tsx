'use client';

type HeaderProps = {
  status: {
    height: number;
    lastBlock: any;
  } | null;
  transactionCount: number;
  validatorCount: number;
  usingMockData?: boolean;
  onRefresh: () => void;
};

export default function Header({ 
  status, 
  transactionCount, 
  validatorCount, 
  usingMockData, 
  onRefresh 
}: HeaderProps) {
  // To get the last block hash
  const getLastBlockHash = (lastBlock: any): string => {
    if (typeof lastBlock === 'string') {
      try {
        return JSON.parse(lastBlock).hash || lastBlock.substring(0, 15);
      } catch (e) {
        return lastBlock.substring(0, 15);
      }
    } else if (lastBlock && typeof lastBlock === 'object') {
      return lastBlock.hash;
    }
    return 'Bilinmiyor';
  };
  
  return (
    <header>
      {usingMockData && (
        <div className="bg-yellow-100 text-yellow-800 px-4 py-2 text-sm text-center mb-4 rounded-lg border border-yellow-300">
          <div className="flex items-center justify-center">
            <svg className="w-5 h-5 mr-2 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>API sunucusuna erişimde sorun yaşanıyor. Şu anda gösterilen veriler güncel olmayabilir.</span>
            <button 
              onClick={onRefresh} 
              className="ml-3 px-2 py-1 bg-yellow-200 hover:bg-yellow-300 rounded-md text-xs font-medium"
            >
              Yenile
            </button>
          </div>
        </div>
      )}
      
      <div className="bg-gradient-to-r from-blue-600 to-indigo-700 text-black rounded-xl shadow-lg p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl text-white font-bold">Confirmix Blockchain</h1>
          <div className="flex items-center">
            <button 
              onClick={onRefresh}
              className="bg-white bg-opacity-20 hover:bg-opacity-30 px-3 py-1 rounded-full text-sm flex items-center"
            >
              <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Yenile
            </button>
          </div>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-white bg-opacity-10 p-4 rounded-lg backdrop-blur-sm">
            <div className="flex items-center">
              <div className="mr-3 bg-blue-500 bg-opacity-40 p-2 rounded-full">
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                </svg>
              </div>
              <div>
                <h3 className="font-medium text-sm text-black">Blockchain Yüksekliği</h3>
                <p className="text-2xl font-bold text-black">{status?.height || '...'}</p>
              </div>
            </div>
          </div>
          
          <div className="bg-white bg-opacity-10 p-4 rounded-lg backdrop-blur-sm">
            <div className="flex items-center">
              <div className="mr-3 bg-indigo-500 bg-opacity-40 p-2 rounded-full">
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div>
                <h3 className="font-medium text-sm text-black">Latest Block</h3>
                <p className="text-sm font-mono truncate max-w-[200px] text-black">
                  {status ? getLastBlockHash(status.lastBlock) : '...'}
                </p>
              </div>
            </div>
          </div>
          
          <div className="bg-white bg-opacity-10 p-4 rounded-lg backdrop-blur-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <div className="mr-3 bg-purple-500 bg-opacity-40 p-2 rounded-full">
                  <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                </div>
                <div>
                  <h3 className="font-medium text-sm text-black">Pending Transactions</h3>
                  <p className="text-2xl font-bold text-black">{transactionCount}</p>
                </div>
              </div>
              <div className="text-right">
                <h3 className="font-medium text-sm text-black">Validators</h3>
                <p className="text-2xl font-bold text-black">{validatorCount}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
} 