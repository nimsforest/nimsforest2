import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'NimsForest Webview',
  description: 'Isometric visualization of NimsForest cluster state',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
