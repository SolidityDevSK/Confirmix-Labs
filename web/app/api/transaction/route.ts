import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function POST(request: Request) {
  try {
    const body = await request.json();
    
    const response = await fetch(`${BACKEND_API_URL}/transaction`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errorData = await response.json();
      return NextResponse.json(
        { error: errorData.error || 'Transfer işlemi başarısız oldu' },
        { status: response.status }
      );
    }

    return NextResponse.json(
      { message: 'Transfer işlemi başarıyla gönderildi' },
      { status: 200 }
    );
  } catch (error) {
    console.error('Transaction endpoint error:', error);
    return NextResponse.json(
      { error: 'Transfer işlemi sırasında bir hata oluştu' },
      { status: 500 }
    );
  }
} 