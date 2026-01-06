import * as Phaser from 'phaser';
import { World, Land, TILE_WIDTH, TILE_HEIGHT, gridToIso, Process } from './types';

export type SelectCallback = (type: 'land' | 'tree' | 'treehouse' | 'nim', id: string, landId?: string) => void;

export class ForestScene extends Phaser.Scene {
  private world: World | null = null;
  private onSelect: SelectCallback | null = null;
  private isDragging = false;
  private lastPointer = { x: 0, y: 0 };

  constructor() {
    super({ key: 'ForestScene' });
  }

  preload() {
    // Create placeholder graphics for tiles and sprites
    this.createPlaceholderAssets();
  }

  private createPlaceholderAssets() {
    // Create land tile (green isometric diamond)
    const landGraphics = this.make.graphics({ x: 0, y: 0 }, false);
    landGraphics.fillStyle(0x228b22, 1); // Forest green
    landGraphics.beginPath();
    landGraphics.moveTo(TILE_WIDTH / 2, 0);
    landGraphics.lineTo(TILE_WIDTH, TILE_HEIGHT / 2);
    landGraphics.lineTo(TILE_WIDTH / 2, TILE_HEIGHT);
    landGraphics.lineTo(0, TILE_HEIGHT / 2);
    landGraphics.closePath();
    landGraphics.fillPath();
    landGraphics.lineStyle(2, 0x1e7a1e, 1);
    landGraphics.strokePath();
    landGraphics.generateTexture('land', TILE_WIDTH, TILE_HEIGHT);
    landGraphics.destroy();

    // Create manaland tile (purple isometric diamond)
    const manaGraphics = this.make.graphics({ x: 0, y: 0 }, false);
    manaGraphics.fillStyle(0x9b59b6, 1); // Purple
    manaGraphics.beginPath();
    manaGraphics.moveTo(TILE_WIDTH / 2, 0);
    manaGraphics.lineTo(TILE_WIDTH, TILE_HEIGHT / 2);
    manaGraphics.lineTo(TILE_WIDTH / 2, TILE_HEIGHT);
    manaGraphics.lineTo(0, TILE_HEIGHT / 2);
    manaGraphics.closePath();
    manaGraphics.fillPath();
    manaGraphics.lineStyle(2, 0x8e44ad, 1);
    manaGraphics.strokePath();
    manaGraphics.generateTexture('manaland', TILE_WIDTH, TILE_HEIGHT);
    manaGraphics.destroy();

    // Create tree sprite (simple triangle tree)
    const treeGraphics = this.make.graphics({ x: 0, y: 0 }, false);
    treeGraphics.fillStyle(0x2d5a27, 1); // Dark green
    treeGraphics.fillTriangle(16, 0, 32, 32, 0, 32);
    treeGraphics.fillStyle(0x3d7a37, 1);
    treeGraphics.fillTriangle(16, 8, 28, 28, 4, 28);
    treeGraphics.fillStyle(0x8b4513, 1); // Brown trunk
    treeGraphics.fillRect(12, 28, 8, 8);
    treeGraphics.generateTexture('tree', 32, 36);
    treeGraphics.destroy();

    // Create treehouse sprite (small house)
    const houseGraphics = this.make.graphics({ x: 0, y: 0 }, false);
    houseGraphics.fillStyle(0x8b4513, 1); // Brown walls
    houseGraphics.fillRect(4, 14, 24, 18);
    houseGraphics.fillStyle(0xc0392b, 1); // Red roof
    houseGraphics.fillTriangle(16, 0, 32, 14, 0, 14);
    houseGraphics.fillStyle(0xf39c12, 1); // Yellow door
    houseGraphics.fillRect(12, 22, 8, 10);
    houseGraphics.generateTexture('treehouse', 32, 32);
    houseGraphics.destroy();

    // Create nim sprite (gear/cog)
    const nimGraphics = this.make.graphics({ x: 0, y: 0 }, false);
    nimGraphics.fillStyle(0x3498db, 1); // Blue
    nimGraphics.fillCircle(16, 16, 12);
    nimGraphics.fillStyle(0x2980b9, 1);
    nimGraphics.fillCircle(16, 16, 6);
    // Gear teeth
    for (let i = 0; i < 8; i++) {
      const angle = (i * Math.PI * 2) / 8;
      const x = 16 + Math.cos(angle) * 14;
      const y = 16 + Math.sin(angle) * 14;
      nimGraphics.fillStyle(0x3498db, 1);
      nimGraphics.fillCircle(x, y, 4);
    }
    nimGraphics.generateTexture('nim', 32, 32);
    nimGraphics.destroy();
  }

  create() {
    // Set background color
    this.cameras.main.setBackgroundColor(0x1a1a2e);

    // Camera drag to pan
    this.input.on('pointerdown', (pointer: Phaser.Input.Pointer) => {
      this.isDragging = false;
      this.lastPointer = { x: pointer.x, y: pointer.y };
    });

    this.input.on('pointermove', (pointer: Phaser.Input.Pointer) => {
      if (pointer.isDown) {
        const dx = pointer.x - this.lastPointer.x;
        const dy = pointer.y - this.lastPointer.y;

        if (Math.abs(dx) > 2 || Math.abs(dy) > 2) {
          this.isDragging = true;
          this.cameras.main.scrollX -= dx;
          this.cameras.main.scrollY -= dy;
        }

        this.lastPointer = { x: pointer.x, y: pointer.y };
      }
    });

    // Mouse wheel zoom
    this.input.on('wheel', (_pointer: Phaser.Input.Pointer, _gameObjects: Phaser.GameObjects.GameObject[], _deltaX: number, deltaY: number) => {
      const cam = this.cameras.main;
      const newZoom = Phaser.Math.Clamp(cam.zoom - deltaY * 0.001, 0.5, 2);
      cam.setZoom(newZoom);
    });

    if (this.world) {
      this.renderWorld();
    }
  }

  setWorld(world: World) {
    this.world = world;
    if (this.scene.isActive()) {
      this.renderWorld();
    }
  }

  setOnSelect(fn: SelectCallback) {
    this.onSelect = fn;
  }

  private renderWorld() {
    // Clear existing objects
    this.children.removeAll();

    if (!this.world || this.world.lands.length === 0) {
      // Show empty state message
      const text = this.add.text(
        this.cameras.main.centerX,
        this.cameras.main.centerY,
        'No Land detected.\nStart your forest daemon to see nodes.',
        {
          fontSize: '18px',
          color: '#888888',
          align: 'center',
        }
      );
      text.setOrigin(0.5);
      text.setScrollFactor(0);
      return;
    }

    // Render each land tile
    for (const land of this.world.lands) {
      this.renderLand(land);
    }

    // Center camera on the grid
    this.centerCamera();
  }

  private renderLand(land: Land) {
    const { x, y } = gridToIso(land.grid_x, land.grid_y);
    const depth = land.grid_x + land.grid_y;

    // Create land tile
    const tileKey = land.is_manaland ? 'manaland' : 'land';
    const tile = this.add.image(x, y, tileKey);
    tile.setDepth(depth);
    tile.setInteractive({ useHandCursor: true });

    // Add hover effect
    tile.on('pointerover', () => {
      tile.setTint(0xdddddd);
    });
    tile.on('pointerout', () => {
      tile.clearTint();
    });

    // Click handler
    tile.on('pointerup', () => {
      if (!this.isDragging) {
        this.onSelect?.('land', land.id);
      }
    });

    // Add label under the tile
    const label = this.add.text(x, y + TILE_HEIGHT / 2 + 4, land.id.substring(0, 8), {
      fontSize: '10px',
      color: '#aaaaaa',
    });
    label.setOrigin(0.5, 0);
    label.setDepth(depth + 0.01);

    // Render processes stacked on the land
    this.renderProcesses(land, x, y, depth);
  }

  private renderProcesses(land: Land, baseX: number, baseY: number, baseDepth: number) {
    let offsetY = -20; // Start above the tile
    const spacing = 24;

    // Combine all processes
    const allProcesses: Array<Process & { processType: 'tree' | 'treehouse' | 'nim' }> = [
      ...land.trees.map(p => ({ ...p, processType: 'tree' as const })),
      ...land.treehouses.map(p => ({ ...p, processType: 'treehouse' as const })),
      ...land.nims.map(p => ({ ...p, processType: 'nim' as const })),
    ];

    for (const proc of allProcesses) {
      const spriteKey = proc.processType;
      const sprite = this.add.image(baseX, baseY + offsetY, spriteKey);
      sprite.setDepth(baseDepth + 0.1);
      sprite.setScale(0.8);
      sprite.setInteractive({ useHandCursor: true });

      // Hover effect
      sprite.on('pointerover', () => {
        sprite.setTint(0xffff00);
        sprite.setScale(0.9);
      });
      sprite.on('pointerout', () => {
        sprite.clearTint();
        sprite.setScale(0.8);
      });

      // Click handler
      sprite.on('pointerup', () => {
        if (!this.isDragging) {
          this.onSelect?.(proc.processType, proc.id, land.id);
        }
      });

      offsetY -= spacing;
    }
  }

  private centerCamera() {
    if (!this.world || this.world.lands.length === 0) return;

    // Find the center of all lands
    let minX = Infinity, maxX = -Infinity;
    let minY = Infinity, maxY = -Infinity;

    for (const land of this.world.lands) {
      const { x, y } = gridToIso(land.grid_x, land.grid_y);
      minX = Math.min(minX, x);
      maxX = Math.max(maxX, x);
      minY = Math.min(minY, y);
      maxY = Math.max(maxY, y);
    }

    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;

    this.cameras.main.centerOn(centerX, centerY);
  }
}
