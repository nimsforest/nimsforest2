'use client';

import { useEffect, useRef } from 'react';
import * as Phaser from 'phaser';
import { ForestScene, SelectCallback } from './ForestScene';
import { World } from './types';

interface Props {
  world: World | null;
  onSelect: SelectCallback;
}

export default function PhaserGame({ world, onSelect }: Props) {
  const gameRef = useRef<Phaser.Game | null>(null);
  const sceneRef = useRef<ForestScene | null>(null);

  useEffect(() => {
    // Initialize Phaser game
    const config: Phaser.Types.Core.GameConfig = {
      type: Phaser.AUTO,
      parent: 'phaser-container',
      width: '100%',
      height: '100%',
      backgroundColor: '#1a1a2e',
      scene: ForestScene,
      scale: {
        mode: Phaser.Scale.RESIZE,
        autoCenter: Phaser.Scale.CENTER_BOTH,
      },
      input: {
        activePointers: 3,
      },
    };

    gameRef.current = new Phaser.Game(config);

    // Get scene reference after it's created
    gameRef.current.events.on('ready', () => {
      sceneRef.current = gameRef.current?.scene.getScene('ForestScene') as ForestScene;
      if (sceneRef.current) {
        sceneRef.current.setOnSelect(onSelect);
        if (world) {
          sceneRef.current.setWorld(world);
        }
      }
    });

    return () => {
      gameRef.current?.destroy(true);
      gameRef.current = null;
      sceneRef.current = null;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Update world when it changes
  useEffect(() => {
    if (sceneRef.current && world) {
      sceneRef.current.setWorld(world);
    }
  }, [world]);

  // Update onSelect callback
  useEffect(() => {
    if (sceneRef.current) {
      sceneRef.current.setOnSelect(onSelect);
    }
  }, [onSelect]);

  return (
    <div
      id="phaser-container"
      className="w-full h-full"
      style={{ minHeight: '400px' }}
    />
  );
}
