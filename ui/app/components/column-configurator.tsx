"use client";

import { useState, useRef, useEffect } from "react";
import type { ColumnConfig, ColumnId } from "../lib/types";
import {
  COLUMN_DEFINITIONS,
  getColumnsByGroup,
} from "../lib/column-config";

interface ColumnConfiguratorProps {
  config: ColumnConfig;
  onSave: (visibleColumns: ColumnId[]) => void;
  onReset: () => void;
}

export function ColumnConfigurator({
  config,
  onSave,
  onReset,
}: ColumnConfiguratorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [tempVisible, setTempVisible] = useState<ColumnId[]>(
    config.visibleColumns
  );
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setTempVisible(config.visibleColumns);
  }, [config.visibleColumns]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen]);

  const handleToggle = (columnId: ColumnId) => {
    setTempVisible((prev) =>
      prev.includes(columnId)
        ? prev.filter((id) => id !== columnId)
        : [...prev, columnId]
    );
  };

  const handleApply = () => {
    onSave(tempVisible);
    setIsOpen(false);
  };

  const handleReset = () => {
    onReset();
    setIsOpen(false);
  };

  const columnsByGroup = getColumnsByGroup();
  const groupLabels: Record<string, string> = {
    basic: "Basic Info",
    image: "Image",
    resources: "Resources",
    network: "Network",
  };

  const visibleCount = tempVisible.length;
  const totalCount = Object.keys(COLUMN_DEFINITIONS).length;

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        type="button"
      >
        Columns ({visibleCount}/{totalCount})
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-80 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg shadow-lg z-50">
          <div className="p-4 border-b border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-sm">Configure Columns</h3>
          </div>

          <div className="p-4 max-h-96 overflow-y-auto">
            {Object.entries(columnsByGroup).map(([groupKey, columns]) => (
              <div key={groupKey} className="mb-4 last:mb-0">
                <div className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase mb-2">
                  {groupLabels[groupKey] || groupKey}
                </div>
                <div className="space-y-2">
                  {columns.map((col) => (
                    <label
                      key={col.id}
                      className="flex items-center gap-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700 p-1 rounded"
                    >
                      <input
                        type="checkbox"
                        checked={tempVisible.includes(col.id)}
                        onChange={() => handleToggle(col.id)}
                        className="rounded border-gray-300 dark:border-gray-600"
                      />
                      <span className="text-sm">{col.label}</span>
                    </label>
                  ))}
                </div>
              </div>
            ))}
          </div>

          <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex gap-2">
            <button
              onClick={handleApply}
              className="flex-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
              type="button"
            >
              Apply
            </button>
            <button
              onClick={handleReset}
              className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              type="button"
            >
              Reset
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
