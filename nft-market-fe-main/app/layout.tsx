import { ThemeProvider } from 'next-themes'
import { Inter } from 'next/font/google'
import "./globals.css"
// import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Metadata } from 'next/types'
import ClientLayout from './client-layout'



const inter = Inter({ subsets: ["latin"] })

export const metadata: Metadata = {
  title: "NFT Marketplace",
  description: "Trade NFT collections",
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    
      <html lang="en" suppressHydrationWarning>
        
        <body className={`${inter.className} min-h-screen`}>
          <ThemeProvider attribute="class"
          defaultTheme="dark"
          enableSystem={true}
          disableTransitionOnChange>

            <ClientLayout>
              {children}
            </ClientLayout>
          
        </ThemeProvider>
        </body>
        
      </html>
    
  )
}

