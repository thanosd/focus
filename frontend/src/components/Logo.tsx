export default function Logo({ className = "w-6 h-6" }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="10" fill="#3b82f6" />
      <circle cx="12" cy="12" r="6.5" fill="#ffffff" />
      <circle cx="12" cy="12" r="3" fill="#3b82f6" />
    </svg>
  );
}
