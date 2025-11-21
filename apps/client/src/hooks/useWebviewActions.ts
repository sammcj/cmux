import { useCallback, useEffect, useRef } from "react";

import { focusWebview } from "@/lib/webview-actions";

interface UseWebviewActionsOptions {
  persistKey: string;
}

interface UseWebviewActionsResult {
  focus: () => Promise<boolean>;
}

export function useWebviewActions({
  persistKey,
}: UseWebviewActionsOptions): UseWebviewActionsResult {
  const persistKeyRef = useRef(persistKey);

  useEffect(() => {
    persistKeyRef.current = persistKey;
  }, [persistKey]);

  const focus = useCallback(() => {
    return focusWebview(persistKeyRef.current);
  }, []);

  return { focus };
}
