'use client';

import { useState, useEffect, useCallback } from 'react';
import dynamic from 'next/dynamic';
import { World, Selected } from './types';
import Sidebar from '../ui/Sidebar';

// Dynamically import PhaserGame to avoid SSR issues
const PhaserGame = dynamic(() => import('./PhaserGame'), {
  ssr: false,
  loading: () => (
    <div className="flex items-center justify-center h-full bg-forest-dark">
      <div className="text-gray-400">Loading game engine...</div>
    </div>
  ),
});

export default function ForestBoard() {
  const [world, setWorld] = useState<World | null>(null);
  const [selected, setSelected] = useState<Selected | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchWorld = useCallback(async () => {
    try {
      setError(null);
      const res = await fetch('/api/viewmodel');
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`);
      }
      const data = await res.json();
      setWorld(data);
    } catch (err) {
      console.error('Failed to fetch world:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchWorld();
  }, [fetchWorld]);

  const handleSelect = useCallback(
    (type: 'land' | 'tree' | 'treehouse' | 'nim', id: string, landId?: string) => {
      setSelected({ type, id, landId });
    },
    []
  );

  const handleClose = useCallback(() => {
    setSelected(null);
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-forest-dark">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-forest-accent mx-auto mb-4"></div>
          <div className="text-gray-400">Loading NimsForest...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-forest-dark">
      {/* Main game area */}
      <div className="flex-1 relative">
        <PhaserGame world={world} onSelect={handleSelect} />

        {/* Top toolbar */}
        <div className="absolute top-4 left-4 flex gap-2">
          <button
            onClick={fetchWorld}
            className="px-4 py-2 bg-forest-accent text-forest-dark font-semibold rounded-lg hover:bg-green-400 transition-colors shadow-lg"
          >
            🔄 Refresh
          </button>
        </div>

        {/* Error banner */}
        {error && (
          <div className="absolute top-4 left-1/2 transform -translate-x-1/2 bg-red-900/90 text-red-100 px-4 py-2 rounded-lg shadow-lg">
            ⚠️ {error}
          </div>
        )}

        {/* Title */}
        <div className="absolute top-4 right-80 text-forest-accent font-bold text-lg">
          🌲 NimsForest
        </div>

        {/* Legend */}
        <div className="absolute bottom-4 left-4 bg-forest-medium/90 rounded-lg p-3 text-sm shadow-lg">
          <div className="font-bold text-forest-accent mb-2">Legend</div>
          <div className="flex flex-col gap-1 text-gray-300">
            <div className="flex items-center gap-2">
              <div className="w-4 h-2 bg-green-700" style={{ clipPath: 'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)' }}></div>
              <span>Land</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-2 bg-purple-600" style={{ clipPath: 'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)' }}></div>
              <span>Manaland (GPU)</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-green-600">🌲</span>
              <span>Tree</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-red-600">🏠</span>
              <span>Treehouse</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-blue-500">⚙️</span>
              <span>Nim</span>
            </div>
          </div>
        </div>

        {/* Controls hint */}
        <div className="absolute bottom-4 right-80 bg-forest-medium/90 rounded-lg p-3 text-sm text-gray-400 shadow-lg">
          <div className="font-bold text-gray-300 mb-1">Controls</div>
          <div>Drag to pan • Scroll to zoom • Click to select</div>
        </div>
      </div>

      {/* Sidebar */}
      <Sidebar world={world} selected={selected} onClose={handleClose} />
    </div>
  );
}
