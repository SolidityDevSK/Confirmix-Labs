import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function GET(
  request: Request, 
  context: { params: Promise<{ address: string }> }
) {
  try {
    // Önce params objesini await et
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
      console.error('Wallet balance endpoint error:', error);
      
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          return NextResponse.json(
            { error: 'Blockchain node yanıt vermedi - zaman aşımı' },
            { status: 504 }
          );
        }
      }

      return NextResponse.json(
        { error: 'Cüzdan bakiyesi alınamadı' },
        { status: 500 }
      );
    }
  } catch (error) {
    console.error('Wallet balance processing error:', error);
    return NextResponse.json(
      { error: 'Cüzdan bakiyesi işlenirken hata oluştu' },
      { status: 500 }
    );
  }
} 