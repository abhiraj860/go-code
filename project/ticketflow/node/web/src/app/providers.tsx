'use client';

import { useState, useRef, type ReactNode } from 'react';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { makeStore, type AppStore } from '@/store';

export function Providers({ children }: { children: ReactNode }) {
  // The store is created per browser session, held in a ref so React's strict
  // mode double-render does not build two of them. Creating it at module scope
  // would share one store across every request on the server — leaking one
  // user's seat selection into another's page.
  const storeRef = useRef<AppStore | null>(null);
  storeRef.current ??= makeStore();

  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // Deliberately conservative defaults. Anything genuinely cacheable
            // sets its own longer staleTime; the dangerous direction here is
            // serving stale seat data, so the default errs toward fresh.
            staleTime: 10_000,
            gcTime: 5 * 60_000,
            retry: 1,
            refetchOnWindowFocus: true,
          },
        },
      }),
  );

  return (
    <Provider store={storeRef.current}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </Provider>
  );
}
