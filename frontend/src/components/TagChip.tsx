import type { Tag } from "@/lib/types";

export default function TagChip({
  tag,
  onRemove,
  size = "sm",
}: {
  tag: Tag;
  onRemove?: () => void;
  size?: "sm" | "md";
}) {
  const sizeCls =
    size === "sm" ? "text-[11px] px-1.5 py-0.5" : "text-xs px-2 py-1";
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full font-medium ${sizeCls}`}
      style={{
        backgroundColor: `${tag.color}1a`,
        color: tag.color,
        border: `1px solid ${tag.color}55`,
      }}
    >
      <span
        className="w-1.5 h-1.5 rounded-full"
        style={{ backgroundColor: tag.color }}
      />
      {tag.name}
      {onRemove && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          className="ml-0.5 hover:opacity-70"
          aria-label={`Remove tag ${tag.name}`}
        >
          ×
        </button>
      )}
    </span>
  );
}
