"use client";

import React, {
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
} from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

export interface ConfirmOptions {
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
}

type ConfirmFn = (options: ConfirmOptions) => Promise<boolean>;

const ConfirmContext = createContext<ConfirmFn | undefined>(undefined);

/** App-wide replacement for window.confirm, rendered as a Dialog. */
export function ConfirmProvider({ children }: { children: React.ReactNode }) {
  const [options, setOptions] = useState<ConfirmOptions | null>(null);
  const resolver = useRef<((v: boolean) => void) | null>(null);

  const confirm = useCallback<ConfirmFn>((opts) => {
    return new Promise<boolean>((resolve) => {
      resolver.current = resolve;
      setOptions(opts);
    });
  }, []);

  const settle = (value: boolean) => {
    resolver.current?.(value);
    resolver.current = null;
    setOptions(null);
  };

  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      <Dialog open={!!options} onOpenChange={(o) => !o && settle(false)}>
        <DialogContent>
          {options && (
            <>
              <DialogTitle>{options.title}</DialogTitle>
              {options.description && (
                <DialogDescription className="mt-1.5">
                  {options.description}
                </DialogDescription>
              )}
              <div className="mt-5 flex justify-end gap-2">
                <Button variant="secondary" onClick={() => settle(false)}>
                  {options.cancelLabel ?? "Cancel"}
                </Button>
                <Button
                  variant={options.destructive ? "danger" : "primary"}
                  onClick={() => settle(true)}
                  autoFocus
                >
                  {options.confirmLabel ?? "Confirm"}
                </Button>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </ConfirmContext.Provider>
  );
}

export function useConfirm(): ConfirmFn {
  const ctx = useContext(ConfirmContext);
  if (!ctx) throw new Error("useConfirm must be used within a ConfirmProvider");
  return ctx;
}
