"use client";

/* eslint-disable @next/next/no-img-element */
import { useState } from "react";

type Size = "sm" | "md" | "lg";

const SIZE_CLASS: Record<Size, string> = {
  sm: "w-6 h-6 text-xs",
  md: "w-8 h-8 text-sm",
  lg: "w-10 h-10 text-base",
};

interface AvatarProps {
  email: string;
  name?: string;
  pictureUrl?: string;
  size?: Size;
  className?: string;
}

export default function Avatar({
  email,
  name,
  pictureUrl,
  size = "md",
  className = "",
}: AvatarProps) {
  const [imgFailed, setImgFailed] = useState(false);
  const initial = (name || email || "?").charAt(0).toUpperCase();
  const sizeClass = SIZE_CLASS[size];

  if (pictureUrl && !imgFailed) {
    return (
      <img
        src={pictureUrl}
        alt=""
        referrerPolicy="no-referrer"
        onError={() => setImgFailed(true)}
        className={`${sizeClass} rounded-full object-cover flex-shrink-0 ${className}`}
      />
    );
  }

  return (
    <div
      className={`${sizeClass} rounded-full bg-blue-600 flex items-center justify-center text-white font-medium flex-shrink-0 ${className}`}
    >
      {initial}
    </div>
  );
}
