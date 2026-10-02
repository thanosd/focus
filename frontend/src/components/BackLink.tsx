import Link from "next/link";
import { ChevronLeft } from "lucide-react";

/** Phone-only "‹ Back" control for one-column-at-a-time navigation. */
export default function BackLink({
  href,
  label,
}: {
  href: string;
  label: string;
}) {
  return (
    <Link
      href={href}
      className="md:hidden inline-flex items-center gap-1 py-2 -ml-1 pr-2 text-sm text-blue-600"
    >
      <ChevronLeft className="w-4 h-4" />
      {label}
    </Link>
  );
}
