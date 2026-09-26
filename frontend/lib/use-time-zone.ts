"use client";

import { useSyncExternalStore } from "react";

// The server doesn't know the viewer's timezone, so the first render (server
// and hydration) uses the app's home zone and the client then switches to the
// browser's zone (R-TIME-2). Identical output for viewers in Thailand.
export const HOME_TIME_ZONE = "Asia/Bangkok";

const noopSubscribe = () => () => {};

export function useTimeZone(): string {
  return useSyncExternalStore(
    noopSubscribe,
    () => Intl.DateTimeFormat().resolvedOptions().timeZone,
    () => HOME_TIME_ZONE,
  );
}
