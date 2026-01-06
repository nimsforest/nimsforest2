// TypeScript types for the NimsForest webview

export interface Process {
  id: string;
  name: string;
  ram_allocated: number;
  type: 'tree' | 'treehouse' | 'nim';
  subjects?: string[];
  script_path?: string;
  ai_enabled?: boolean;
  model?: string;
}

export interface Land {
  id: string;
  hostname: string;
  ram_total: number;
  ram_allocated: number;
  cpu_cores: number;
  cpu_freq_ghz: number;
  gpu_vram?: number;
  gpu_tflops?: number;
  occupancy: number;
  is_manaland: boolean;
  grid_x: number;
  grid_y: number;
  trees: Process[];
  treehouses: Process[];
  nims: Process[];
}

export interface Summary {
  land_count: number;
  manaland_count: number;
  tree_count: number;
  treehouse_count: number;
  nim_count: number;
  total_ram: number;
  ram_allocated: number;
  occupancy: number;
}

export interface World {
  lands: Land[];
  summary: Summary;
}

export interface Selected {
  type: 'land' | 'tree' | 'treehouse' | 'nim';
  id: string;
  landId?: string;
}

// Isometric tile dimensions
export const TILE_WIDTH = 88;
export const TILE_HEIGHT = 44;

// Convert grid coordinates to isometric screen coordinates
export function gridToIso(gridX: number, gridY: number): { x: number; y: number } {
  return {
    x: (gridX - gridY) * (TILE_WIDTH / 2),
    y: (gridX + gridY) * (TILE_HEIGHT / 2),
  };
}

// Convert screen coordinates to grid coordinates
export function isoToGrid(screenX: number, screenY: number): { gridX: number; gridY: number } {
  const gridX = (screenX / (TILE_WIDTH / 2) + screenY / (TILE_HEIGHT / 2)) / 2;
  const gridY = (screenY / (TILE_HEIGHT / 2) - screenX / (TILE_WIDTH / 2)) / 2;
  return { gridX: Math.floor(gridX), gridY: Math.floor(gridY) };
}

// Format bytes to human-readable string
export function formatBytes(bytes: number): string {
  if (bytes >= 1e12) return (bytes / 1e12).toFixed(1) + 'TB';
  if (bytes >= 1e9) return (bytes / 1e9).toFixed(1) + 'GB';
  if (bytes >= 1e6) return (bytes / 1e6).toFixed(0) + 'MB';
  if (bytes >= 1e3) return (bytes / 1e3).toFixed(0) + 'KB';
  return bytes + 'B';
}
