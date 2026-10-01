import type { Modifier } from "@dnd-kit/core";

/** Keep sortable rows on their column while dragging. */
const restrictToVerticalAxis: Modifier = ({ transform }) => ({
  ...transform,
  x: 0,
});

export const restrictToVerticalAxisIfAvailable = [restrictToVerticalAxis];
