import { useEffect, useMemo, useRef, useState } from "react";

import { ChevronDown } from "lucide-react";

type InlineSelectOption = {
  label: string;
  value: number | string;
};

type InlineSelectProps = {
  value: number | string;
  options: InlineSelectOption[];
  disabled?: boolean;
  onChange: (value: number | string) => void;
};

export function InlineSelect({ value, options, disabled = false, onChange }: InlineSelectProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement | null>(null);

  const selected = useMemo(() => {
    return options.find((option) => String(option.value) === String(value)) ?? null;
  }, [options, value]);

  useEffect(() => {
    function handlePointerDown(event: MouseEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }

    window.addEventListener("mousedown", handlePointerDown);
    window.addEventListener("keydown", handleEscape);

    return () => {
      window.removeEventListener("mousedown", handlePointerDown);
      window.removeEventListener("keydown", handleEscape);
    };
  }, []);

  return (
    <div className="admin-inline-select" ref={rootRef}>
      <button
        type="button"
        className="admin-inline-select-trigger"
        data-open={open ? "true" : "false"}
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen((prev) => !prev)}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown" || event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            setOpen(true);
          }
        }}
      >
        <span className="admin-inline-select-label">{selected?.label ?? String(value)}</span>
        <ChevronDown className="admin-inline-select-icon" strokeWidth={1.8} />
      </button>

      {open ? (
        <div className="admin-inline-select-menu" role="listbox">
          {options.map((option) => {
            const isSelected = String(option.value) === String(value);

            return (
              <button
                key={String(option.value)}
                type="button"
                role="option"
                aria-selected={isSelected}
                className="admin-inline-select-option"
                data-selected={isSelected ? "true" : "false"}
                onClick={() => {
                  onChange(option.value);
                  setOpen(false);
                }}
              >
                {option.label}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
