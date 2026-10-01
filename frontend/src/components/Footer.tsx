import Image from "next/image";

export default function Footer() {
  return (
    <footer className="border-t border-gray-200 bg-white py-5 px-6">
      <a
        href="https://cosmicteacups.com"
        target="_blank"
        rel="noopener noreferrer"
        className="group flex items-center justify-center gap-3 text-sm text-gray-600 transition-opacity hover:opacity-90"
      >
        <span>Crafted with love (and a lot of AI) at Cosmic Teacups</span>
        <Image
          src="/cosmic/cosmic-teacups-logo.png"
          alt="Cosmic Teacups"
          width={56}
          height={56}
          className="h-12 w-auto rounded"
        />
      </a>
    </footer>
  );
}
