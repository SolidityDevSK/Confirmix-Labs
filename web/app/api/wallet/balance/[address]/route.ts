import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function GET(
  request: Request, 
  context: { params: Promise<{ address: string }> }
) {
  try {
    // First await the params object
    const { address } = await context.params;
    console.log('Wallet balance endpoint called for address:', address);

    // Timeout için AbortController
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 30000);

    try {
      const response = await fetch(`${BACKEND_API_URL}/wallet/balance/${address}`, {
        signal: controller.signal
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const balance = await response.text();
      console.log('Backend balance raw response:', balance);
      
      return NextResponse.json({ balance: parseInt(balance) });
    } catch (error) {
      console.error('Error fetching wallet balance:', error);
      
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          return NextResponse.json(
            { error: 'Backend server did not respond (timeout)' },
            { status: 504 }
          );
        }

        if (error.cause && typeof error.cause === 'object' && 'code' in error.cause) {
          if (error.cause.code === 'ECONNREFUSED') {
            return NextResponse.json(
              { error: 'Could not connect to backend server. Please make sure the server is running.' },
              { status: 503 }
            );
          }
        }
        
        return NextResponse.json(
          { error: 'Could not retrieve balance information: ' + error.message },
          { status: 500 }
        );
      } else {
        return NextResponse.json(
          { error: 'Could not retrieve balance information: Unknown error' },
          { status: 500 }
        );
      }
    }
  } catch (error) {
    console.error('Wallet balance processing error:', error);
    return NextResponse.json(
      { error: 'Error processing wallet balance' },
      { status: 500 }
    );
  }
} 