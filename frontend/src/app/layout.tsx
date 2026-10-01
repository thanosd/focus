import type { Metadata } from "next";
import { Plus_Jakarta_Sans } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "@/contexts/AuthContext";
import { CountsProvider } from "@/contexts/CountsContext";
import { ToastProvider } from "@/contexts/ToastContext";
import { ConfirmProvider } from "@/contexts/ConfirmContext";
import Sidebar from "@/components/Sidebar";
import AppHeader from "@/components/AppHeader";
import Footer from "@/components/Footer";

const focusSans = Plus_Jakarta_Sans({
  subsets: ["latin"],
  variable: "--font-focus-sans",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Focus",
  description: "Projects, tasks, tags and reviews",
  icons: {
    icon: [
      { url: "/favicons/favicon.ico", sizes: "any" },
      { url: "/favicons/favicon.svg", type: "image/svg+xml" },
      { url: "/favicons/favicon-32x32.png", sizes: "32x32", type: "image/png" },
      { url: "/favicons/favicon-16x16.png", sizes: "16x16", type: "image/png" },
    ],
    apple: [
      {
        url: "/favicons/apple-touch-icon.png",
        sizes: "180x180",
        type: "image/png",
      },
    ],
    other: [{ rel: "manifest", url: "/favicons/site.webmanifest" }],
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={focusSans.variable}>
        <AuthProvider>
          <CountsProvider>
            <ToastProvider>
              <ConfirmProvider>
                <div className="min-h-screen flex flex-col bg-gray-50">
                  <AppHeader />
                  <div className="flex flex-1">
                    <Sidebar />
                    <main className="flex-1 min-w-0 flex flex-col">
                      {children}
                    </main>
                  </div>
                  <Footer />
                </div>
              </ConfirmProvider>
            </ToastProvider>
          </CountsProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
