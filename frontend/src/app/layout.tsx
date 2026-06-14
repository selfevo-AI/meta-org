import type { Metadata } from 'next'
import { LanguageProvider } from '@/lib/i18n'
import '@xyflow/react/dist/style.css'
import './globals.css'

export const metadata: Metadata = {
  title: 'Harness Organization System',
  description: 'Self-evolving organizational management platform',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body className="min-h-screen bg-slate-50 text-slate-900">
        <LanguageProvider>{children}</LanguageProvider>
      </body>
    </html>
  )
}
