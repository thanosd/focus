const base = {
  fill: "none",
  viewBox: "0 0 24 24",
  stroke: "currentColor",
  strokeWidth: 2,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
};

export function FlagIcon({
  className = "w-4 h-4",
  filled = false,
}: {
  className?: string;
  filled?: boolean;
}) {
  return (
    <svg
      className={className}
      {...base}
      fill={filled ? "currentColor" : "none"}
      aria-hidden="true"
    >
      <path d="M5 21V4a1 1 0 011-1h9l1 2h4v10h-6l-1-2H5" />
    </svg>
  );
}

export function ClockIcon({ className = "w-3.5 h-3.5" }: { className?: string }) {
  return (
    <svg className={className} {...base} aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 2" />
    </svg>
  );
}

export function CalendarIcon({
  className = "w-3.5 h-3.5",
}: {
  className?: string;
}) {
  return (
    <svg className={className} {...base} aria-hidden="true">
      <rect x="3" y="5" width="18" height="16" rx="2" />
      <path d="M3 10h18M8 3v4M16 3v4" />
    </svg>
  );
}

export function RepeatIcon({ className = "w-3.5 h-3.5" }: { className?: string }) {
  return (
    <svg className={className} {...base} aria-hidden="true">
      <path d="M17 2l4 4-4 4" />
      <path d="M3 11V9a4 4 0 014-4h14" />
      <path d="M7 22l-4-4 4-4" />
      <path d="M21 13v2a4 4 0 01-4 4H3" />
    </svg>
  );
}

export function CloseIcon({ className = "w-5 h-5" }: { className?: string }) {
  return (
    <svg className={className} {...base} aria-hidden="true">
      <path d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}

export function PlusIcon({ className = "w-4 h-4" }: { className?: string }) {
  return (
    <svg className={className} {...base} aria-hidden="true">
      <path d="M12 5v14M5 12h14" />
    </svg>
  );
}

export function CheckIcon({ className = "w-3.5 h-3.5" }: { className?: string }) {
  return (
    <svg className={className} {...base} strokeWidth={3} aria-hidden="true">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

export function FolderIcon({ className = "w-3.5 h-3.5" }: { className?: string }) {
  return (
    <svg className={className} {...base} aria-hidden="true">
      <path d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" />
    </svg>
  );
}
