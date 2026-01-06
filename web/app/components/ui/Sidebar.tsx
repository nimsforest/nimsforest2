'use client';

import { World, Selected, Land, Process, formatBytes } from '../game/types';

interface Props {
  world: World | null;
  selected: Selected | null;
  onClose: () => void;
}

export default function Sidebar({ world, selected, onClose }: Props) {
  if (!world) return null;

  // Find the selected land
  const selectedLand =
    selected?.type === 'land'
      ? world.lands.find((l) => l.id === selected.id)
      : selected?.landId
      ? world.lands.find((l) => l.id === selected.landId)
      : null;

  // Find the selected process
  const selectedProcess =
    selected && selected.type !== 'land' && selectedLand
      ? findProcess(selectedLand, selected.type, selected.id)
      : null;

  return (
    <div className="w-80 bg-forest-medium text-white p-4 overflow-y-auto border-l border-forest-light/30">
      {/* World Summary */}
      <h2 className="text-xl font-bold mb-4 text-forest-accent">World Summary</h2>

      <div className="mb-6 text-sm space-y-1 text-gray-300">
        <StatRow
          label="Land"
          value={`${world.summary.land_count} (${world.summary.manaland_count} mana)`}
        />
        <StatRow label="Trees" value={world.summary.tree_count.toString()} />
        <StatRow label="Treehouses" value={world.summary.treehouse_count.toString()} />
        <StatRow label="Nims" value={world.summary.nim_count.toString()} />
        <div className="border-t border-forest-light/20 my-2"></div>
        <StatRow label="Total RAM" value={formatBytes(world.summary.total_ram)} />
        <StatRow label="Allocated" value={formatBytes(world.summary.ram_allocated)} />
        <OccupancyBar occupancy={world.summary.occupancy} />
      </div>

      {/* Selected Entity Details */}
      {selected && (
        <div className="border-t border-forest-light/30 pt-4">
          <div className="flex justify-between items-center mb-3">
            <h3 className="font-bold text-forest-accent capitalize">
              {selected.type}: {selected.id.substring(0, 12)}
              {selected.id.length > 12 ? '...' : ''}
            </h3>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white transition-colors text-xl leading-none"
            >
              ×
            </button>
          </div>

          {selected.type === 'land' && selectedLand && (
            <LandDetails land={selectedLand} />
          )}

          {selected.type !== 'land' && selectedProcess && (
            <ProcessDetails process={selectedProcess} type={selected.type} />
          )}
        </div>
      )}

      {/* Lands List */}
      {!selected && world.lands.length > 0 && (
        <div className="border-t border-forest-light/30 pt-4">
          <h3 className="font-bold text-forest-accent mb-3">All Land</h3>
          <div className="space-y-2">
            {world.lands.map((land) => (
              <LandCard key={land.id} land={land} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function StatRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-gray-400">{label}</span>
      <span>{value}</span>
    </div>
  );
}

function OccupancyBar({ occupancy }: { occupancy: number }) {
  const color =
    occupancy < 50 ? 'bg-green-500' : occupancy < 80 ? 'bg-yellow-500' : 'bg-red-500';

  return (
    <div className="mt-2">
      <div className="flex justify-between text-xs mb-1">
        <span className="text-gray-400">Occupancy</span>
        <span>{occupancy.toFixed(0)}%</span>
      </div>
      <div className="w-full bg-gray-700 rounded-full h-2">
        <div
          className={`${color} h-2 rounded-full transition-all duration-300`}
          style={{ width: `${Math.min(100, occupancy)}%` }}
        ></div>
      </div>
    </div>
  );
}

function LandDetails({ land }: { land: Land }) {
  const processCount = land.trees.length + land.treehouses.length + land.nims.length;

  return (
    <div className="text-sm space-y-2 text-gray-300">
      <StatRow label="ID" value={land.id} />
      <StatRow label="Hostname" value={land.hostname || 'N/A'} />
      <StatRow label="Type" value={land.is_manaland ? '🟣 Manaland' : '🟢 Land'} />
      <div className="border-t border-forest-light/20 my-2"></div>
      <StatRow
        label="RAM"
        value={`${formatBytes(land.ram_allocated)} / ${formatBytes(land.ram_total)}`}
      />
      <StatRow label="CPU" value={`${land.cpu_cores} cores @ ${land.cpu_freq_ghz.toFixed(1)} GHz`} />
      {land.gpu_vram && land.gpu_vram > 0 && (
        <>
          <StatRow label="GPU VRAM" value={formatBytes(land.gpu_vram)} />
          {land.gpu_tflops && <StatRow label="GPU TFLOPs" value={land.gpu_tflops.toFixed(1)} />}
        </>
      )}
      <OccupancyBar occupancy={land.occupancy} />
      <div className="border-t border-forest-light/20 my-2"></div>
      <StatRow label="Processes" value={processCount.toString()} />
      {land.trees.length > 0 && (
        <div className="pl-4 text-xs text-gray-400">
          {land.trees.length} tree{land.trees.length !== 1 ? 's' : ''}
        </div>
      )}
      {land.treehouses.length > 0 && (
        <div className="pl-4 text-xs text-gray-400">
          {land.treehouses.length} treehouse{land.treehouses.length !== 1 ? 's' : ''}
        </div>
      )}
      {land.nims.length > 0 && (
        <div className="pl-4 text-xs text-gray-400">
          {land.nims.length} nim{land.nims.length !== 1 ? 's' : ''}
        </div>
      )}
    </div>
  );
}

function ProcessDetails({
  process,
  type,
}: {
  process: Process;
  type: 'tree' | 'treehouse' | 'nim';
}) {
  const icon = type === 'tree' ? '🌲' : type === 'treehouse' ? '🏠' : '⚙️';

  return (
    <div className="text-sm space-y-2 text-gray-300">
      <StatRow label="Type" value={`${icon} ${type}`} />
      <StatRow label="ID" value={process.id} />
      <StatRow label="Name" value={process.name || 'N/A'} />
      <div className="border-t border-forest-light/20 my-2"></div>
      <StatRow label="RAM" value={formatBytes(process.ram_allocated)} />
      {process.subjects && process.subjects.length > 0 && (
        <div>
          <span className="text-gray-400">Subjects:</span>
          <div className="pl-2 text-xs mt-1 space-y-1">
            {process.subjects.map((subj, i) => (
              <div key={i} className="bg-forest-dark px-2 py-1 rounded font-mono">
                {subj}
              </div>
            ))}
          </div>
        </div>
      )}
      {process.script_path && (
        <StatRow label="Script" value={process.script_path} />
      )}
      {process.ai_enabled && (
        <>
          <StatRow label="AI Enabled" value="Yes" />
          {process.model && <StatRow label="Model" value={process.model} />}
        </>
      )}
    </div>
  );
}

function LandCard({ land }: { land: Land }) {
  const processCount = land.trees.length + land.treehouses.length + land.nims.length;

  return (
    <div className="bg-forest-dark rounded-lg p-3 text-sm">
      <div className="flex justify-between items-center mb-2">
        <span className="font-semibold">
          {land.is_manaland ? '🟣' : '🟢'} {land.id.substring(0, 12)}...
        </span>
        <span className="text-gray-400 text-xs">{processCount} proc</span>
      </div>
      <div className="text-xs text-gray-400">
        {formatBytes(land.ram_allocated)} / {formatBytes(land.ram_total)}
      </div>
      <OccupancyBar occupancy={land.occupancy} />
    </div>
  );
}

function findProcess(
  land: Land,
  type: 'tree' | 'treehouse' | 'nim',
  id: string
): Process | null {
  const processes = type === 'tree' ? land.trees : type === 'treehouse' ? land.treehouses : land.nims;
  return processes.find((p) => p.id === id) || null;
}
