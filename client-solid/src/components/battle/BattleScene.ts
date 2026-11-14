import Phaser from 'phaser';

import type {
  ApiBattleEvent,
  ApiTacticalBattle,
  ApiTacticalShip,
} from '../../lib/types/battle';
import { canShipFire, currentBattleShip, isMissileFire } from '../../lib/types/battle';

export type BattleIntent =
  | { type: 'move'; x: number; y: number }
  | { type: 'fire'; targetId: number };

export type BattleSceneConfig = {
  battle: ApiTacticalBattle;
  playerId: number;
  colorByOwner: Record<number, string>;
  /** The player clicked a square / an enemy while their ship is acting. */
  onIntent: (intent: BattleIntent) => void;
  /** All queued animations are done and the scene mirrors `battle`. */
  onSettled: (battle: ApiTacticalBattle) => void;
  /** Hover/selection feedback for the HUD panels. */
  onInspectShip: (ship: ApiTacticalShip | null) => void;
};

const GRID_MARGIN_X = 40;
const GRID_MARGIN_TOP = 28;
const GRID_MARGIN_BOTTOM = 16;
const MOVE_MS_PER_SQUARE = 110;
const MISSILE_MS_PER_SQUARE = 45;
const MISSILE_MIN_MS = 220;

type ShipVisual = {
  container: Phaser.GameObjects.Container;
  hull: Phaser.GameObjects.Graphics;
  hpBar: Phaser.GameObjects.Graphics;
  shipId: number;
};

export class BattleScene extends Phaser.Scene {
  private config!: BattleSceneConfig;
  private battle!: ApiTacticalBattle;
  private lastSeq = 0;

  private cellSize = 48;
  private originX = 0;
  private originY = 0;

  private gridGfx!: Phaser.GameObjects.Graphics;
  private rangeGfx!: Phaser.GameObjects.Graphics;
  private cursorGfx!: Phaser.GameObjects.Graphics;
  private activeRing!: Phaser.GameObjects.Graphics;
  private fxLayer!: Phaser.GameObjects.Container;
  private visuals = new Map<number, ShipVisual>();

  private queue: ApiBattleEvent[] = [];
  private animating = false;
  private activeRingTween: Phaser.Tweens.Tween | null = null;

  constructor() {
    super('battle');
  }

  init(config: BattleSceneConfig) {
    this.config = config;
    this.battle = config.battle;
    // The initial state already reflects every event that happened during
    // creation (placement, opening AI activations); render it directly and
    // only animate events that arrive after this point.
    this.lastSeq = config.battle.events.length
      ? config.battle.events[config.battle.events.length - 1].seq
      : 0;
  }

  create() {
    this.computeLayout();
    this.paintStarfield();

    this.gridGfx = this.add.graphics();
    this.rangeGfx = this.add.graphics();
    this.cursorGfx = this.add.graphics();
    this.activeRing = this.add.graphics();
    this.paintGrid();

    for (const ship of this.battle.ships) {
      if (!ship.destroyed) {
        this.visuals.set(ship.id, this.createShipVisual(ship));
      }
    }
    this.fxLayer = this.add.container(0, 0);

    this.input.on('pointermove', (pointer: Phaser.Input.Pointer) =>
      this.handlePointerMove(pointer),
    );
    this.input.on('pointerdown', (pointer: Phaser.Input.Pointer) =>
      this.handlePointerDown(pointer),
    );
    this.scale.on('resize', () => this.relayout());

    this.refreshOverlays();
    this.config.onSettled(this.battle);
  }

  /** Entry point for fresh server state: animates events newer than lastSeq. */
  applyBattle(battle: ApiTacticalBattle) {
    this.battle = battle;
    const fresh = battle.events.filter((event) => event.seq > this.lastSeq);
    if (fresh.length > 0) {
      this.lastSeq = fresh[fresh.length - 1].seq;
      this.queue.push(...fresh);
    }
    if (!this.animating) {
      this.drainQueue();
    }
  }

  private shipById(id: number | undefined): ApiTacticalShip | null {
    return this.battle.ships.find((ship) => ship.id === id) ?? null;
  }

  private computeLayout() {
    const width = this.scale.width;
    const height = this.scale.height;
    const cellW = (width - GRID_MARGIN_X * 2) / this.battle.grid_width;
    const cellH =
      (height - GRID_MARGIN_TOP - GRID_MARGIN_BOTTOM) / this.battle.grid_height;
    this.cellSize = Math.floor(Math.min(cellW, cellH));
    this.originX =
      Math.floor((width - this.cellSize * this.battle.grid_width) / 2);
    this.originY =
      GRID_MARGIN_TOP +
      Math.floor(
        (height -
          GRID_MARGIN_TOP -
          GRID_MARGIN_BOTTOM -
          this.cellSize * this.battle.grid_height) /
          2,
      );
  }

  private relayout() {
    this.computeLayout();
    this.paintGrid();
    for (const ship of this.battle.ships) {
      const visual = this.visuals.get(ship.id);
      if (visual) {
        const { x, y } = this.cellCenter(ship.x, ship.y);
        visual.container.setPosition(x, y);
      }
    }
    this.refreshOverlays();
  }

  private cellCenter(cellX: number, cellY: number) {
    return {
      x: this.originX + cellX * this.cellSize + this.cellSize / 2,
      y: this.originY + cellY * this.cellSize + this.cellSize / 2,
    };
  }

  private cellAt(pointer: Phaser.Input.Pointer): { x: number; y: number } | null {
    const x = Math.floor((pointer.x - this.originX) / this.cellSize);
    const y = Math.floor((pointer.y - this.originY) / this.cellSize);
    if (x < 0 || x >= this.battle.grid_width || y < 0 || y >= this.battle.grid_height) {
      return null;
    }
    return { x, y };
  }

  private livingShipAt(x: number, y: number): ApiTacticalShip | null {
    return (
      this.battle.ships.find(
        (ship) => !ship.destroyed && ship.x === x && ship.y === y,
      ) ?? null
    );
  }

  private paintStarfield() {
    const gfx = this.add.graphics();
    const rng = new Phaser.Math.RandomDataGenerator(['ssg-battle']);
    gfx.fillStyle(0x05060d, 1);
    gfx.fillRect(0, 0, this.scale.width, this.scale.height);
    for (let i = 0; i < 260; i++) {
      const brightness = rng.realInRange(0.25, 1);
      const tint = rng.pick([0xffffff, 0xcfd8ff, 0xffe9c9, 0x9fb8ff]);
      gfx.fillStyle(tint, brightness * 0.9);
      gfx.fillCircle(
        rng.between(0, 1600),
        rng.between(0, 1000),
        rng.realInRange(0.4, brightness > 0.85 ? 1.8 : 1.1),
      );
    }
  }

  private paintGrid() {
    const gfx = this.gridGfx;
    gfx.clear();
    gfx.lineStyle(1, 0x3b4d7a, 0.28);
    const w = this.battle.grid_width;
    const h = this.battle.grid_height;
    for (let x = 0; x <= w; x++) {
      gfx.lineBetween(
        this.originX + x * this.cellSize,
        this.originY,
        this.originX + x * this.cellSize,
        this.originY + h * this.cellSize,
      );
    }
    for (let y = 0; y <= h; y++) {
      gfx.lineBetween(
        this.originX,
        this.originY + y * this.cellSize,
        this.originX + w * this.cellSize,
        this.originY + y * this.cellSize,
      );
    }
  }

  private ownerColor(ownerId: number): number {
    const hex = this.config.colorByOwner[ownerId];
    if (!hex) {
      return ownerId === this.config.playerId ? 0x69d2ff : 0xff7a6b;
    }
    return Phaser.Display.Color.HexStringToColor(hex).color;
  }

  /** Procedural hull: an angular silhouette scaled by hull size, tinted per owner. */
  private createShipVisual(ship: ApiTacticalShip): ShipVisual {
    const { x, y } = this.cellCenter(ship.x, ship.y);
    const container = this.add.container(x, y);
    const hull = this.add.graphics();
    const facingRight = ship.x < this.battle.grid_width / 2;
    this.drawHull(hull, ship, facingRight);
    const hpBar = this.add.graphics();
    container.add([hull, hpBar]);
    container.setDepth(10);
    this.visuals.set(ship.id, { container, hull, hpBar, shipId: ship.id });
    this.paintHpBar(ship);
    return this.visuals.get(ship.id)!;
  }

  private hullScale(ship: ApiTacticalShip) {
    const rating = Math.max(1, Math.min(8, ship.size_rating || 3));
    return (0.45 + rating * 0.09) * (this.cellSize / 48);
  }

  private drawHull(
    gfx: Phaser.GameObjects.Graphics,
    ship: ApiTacticalShip,
    facingRight: boolean,
  ) {
    const color = this.ownerColor(ship.owner_id);
    const s = this.hullScale(ship);
    const dir = facingRight ? 1 : -1;
    gfx.clear();

    // engine glow
    gfx.fillStyle(0xffb066, 0.85);
    gfx.fillCircle(-20 * s * dir, 0, 4.5 * s);
    gfx.fillStyle(0xfff2cc, 0.9);
    gfx.fillCircle(-20 * s * dir, 0, 2.2 * s);

    // main hull silhouette
    gfx.fillStyle(0x9aa7bd, 1);
    gfx.beginPath();
    gfx.moveTo(24 * s * dir, 0);
    gfx.lineTo(8 * s * dir, -7 * s);
    gfx.lineTo(-10 * s * dir, -9 * s);
    gfx.lineTo(-18 * s * dir, -4 * s);
    gfx.lineTo(-18 * s * dir, 4 * s);
    gfx.lineTo(-10 * s * dir, 9 * s);
    gfx.lineTo(8 * s * dir, 7 * s);
    gfx.closePath();
    gfx.fillPath();

    // owner-coloured trim
    gfx.fillStyle(color, 1);
    gfx.fillRect(Math.min(-14 * s * dir, 2 * s * dir), -3 * s, 16 * s, 6 * s);
    gfx.fillStyle(0x2c3548, 1);
    gfx.fillCircle(12 * s * dir, 0, 2.6 * s);
  }

  private paintHpBar(ship: ApiTacticalShip) {
    const visual = this.visuals.get(ship.id);
    if (!visual) return;
    const gfx = visual.hpBar;
    gfx.clear();
    const width = this.cellSize * 0.72;
    const yOff = this.cellSize * 0.38;
    const structureRatio =
      ship.status.max_structure_hp > 0
        ? ship.status.current_structure_hp / ship.status.max_structure_hp
        : 0;
    const armorRatio =
      ship.status.max_armor_hp > 0
        ? ship.status.current_armor_hp / ship.status.max_armor_hp
        : 0;
    gfx.fillStyle(0x10141f, 0.85);
    gfx.fillRect(-width / 2, yOff, width, 5);
    gfx.fillStyle(0x53e08c, 1);
    gfx.fillRect(-width / 2, yOff, width * Math.max(0, structureRatio), 2.4);
    gfx.fillStyle(0xd9b84a, 1);
    gfx.fillRect(-width / 2, yOff + 2.6, width * Math.max(0, armorRatio), 2.4);
  }

  /** Highlights: active ship ring + reachable squares when it is ours. */
  private refreshOverlays() {
    const active = currentBattleShip(this.battle);
    this.rangeGfx.clear();
    this.activeRing.clear();
    this.activeRingTween?.stop();
    this.activeRingTween = null;
    this.activeRing.setAlpha(1);

    if (!active) return;
    const center = this.cellCenter(active.x, active.y);
    const mine = active.owner_id === this.config.playerId;
    this.activeRing.lineStyle(2, mine ? 0x7df0a8 : 0xff8a7d, 0.95);
    this.activeRing.strokeCircle(center.x, center.y, this.cellSize * 0.46);
    this.activeRingTween = this.tweens.add({
      targets: this.activeRing,
      alpha: 0.35,
      duration: 520,
      yoyo: true,
      repeat: -1,
    });

    if (!mine || this.animating) return;
    this.rangeGfx.fillStyle(0x5f86ff, 0.13);
    this.rangeGfx.lineStyle(1, 0x5f86ff, 0.25);
    for (let x = 0; x < this.battle.grid_width; x++) {
      for (let y = 0; y < this.battle.grid_height; y++) {
        const dist = Math.max(Math.abs(x - active.x), Math.abs(y - active.y));
        if (dist === 0 || dist > active.moves_left) continue;
        if (this.livingShipAt(x, y)) continue;
        this.rangeGfx.fillRect(
          this.originX + x * this.cellSize + 1,
          this.originY + y * this.cellSize + 1,
          this.cellSize - 2,
          this.cellSize - 2,
        );
      }
    }
  }

  private handlePointerMove(pointer: Phaser.Input.Pointer) {
    this.cursorGfx.clear();
    const cell = this.cellAt(pointer);
    if (!cell) {
      this.config.onInspectShip(null);
      return;
    }
    const hovered = this.livingShipAt(cell.x, cell.y);
    this.config.onInspectShip(hovered);

    if (this.animating || this.battle.status !== 'active') return;
    const active = currentBattleShip(this.battle);
    if (!active || active.owner_id !== this.config.playerId) return;

    const isEnemy = hovered && hovered.owner_id !== this.config.playerId;
    const canShoot = isEnemy && canShipFire(active);
    const dist = Math.max(Math.abs(cell.x - active.x), Math.abs(cell.y - active.y));
    const canMove = !hovered && dist > 0 && dist <= active.moves_left;
    if (!canShoot && !canMove) return;

    this.cursorGfx.lineStyle(2, canShoot ? 0xff5d4f : 0x7df0a8, 0.9);
    this.cursorGfx.strokeRect(
      this.originX + cell.x * this.cellSize + 1,
      this.originY + cell.y * this.cellSize + 1,
      this.cellSize - 2,
      this.cellSize - 2,
    );
  }

  private handlePointerDown(pointer: Phaser.Input.Pointer) {
    if (pointer.button !== 0) return;
    if (this.animating || this.battle.status !== 'active') return;
    const active = currentBattleShip(this.battle);
    if (!active || active.owner_id !== this.config.playerId) return;
    const cell = this.cellAt(pointer);
    if (!cell) return;

    const clicked = this.livingShipAt(cell.x, cell.y);
    if (clicked && clicked.owner_id !== this.config.playerId) {
      if (canShipFire(active)) {
        this.config.onIntent({ type: 'fire', targetId: clicked.id });
      }
      return;
    }
    if (!clicked) {
      const dist = Math.max(Math.abs(cell.x - active.x), Math.abs(cell.y - active.y));
      if (dist > 0 && dist <= active.moves_left) {
        this.config.onIntent({ type: 'move', x: cell.x, y: cell.y });
      }
    }
  }

  // ----- event animation queue -----

  private drainQueue() {
    const event = this.queue.shift();
    if (!event) {
      this.animating = false;
      this.syncToState();
      this.refreshOverlays();
      this.config.onSettled(this.battle);
      return;
    }
    this.animating = true;
    this.rangeGfx.clear();
    this.cursorGfx.clear();
    this.animateEvent(event, () => this.drainQueue());
  }

  private animateEvent(event: ApiBattleEvent, done: () => void) {
    switch (event.type) {
      case 'move':
        return this.animateMove(event, done);
      case 'fire':
        return this.animateFire(event, done);
      case 'destroyed':
        return this.animateDestroyed(event, done);
      case 'round_start':
        return this.announce(`ROUND ${event.round ?? '?'}`, 0xcfd8ff, done);
      case 'finished':
        return this.animateFinished(event, done);
      case 'activation':
        this.animateActivation(event);
        return done();
      default:
        return done();
    }
  }

  private animateActivation(event: ApiBattleEvent) {
    // Spotlight follows the acting ship even mid-queue.
    const ship = this.shipById(event.ship_id);
    if (!ship) return;
    this.activeRing.clear();
    this.activeRingTween?.stop();
    this.activeRing.setAlpha(1);
    const center = this.cellCenter(ship.x, ship.y);
    const mine = ship.owner_id === this.config.playerId;
    this.activeRing.lineStyle(2, mine ? 0x7df0a8 : 0xff8a7d, 0.95);
    this.activeRing.strokeCircle(center.x, center.y, this.cellSize * 0.46);
  }

  private animateMove(event: ApiBattleEvent, done: () => void) {
    const visual = event.ship_id ? this.visuals.get(event.ship_id) : undefined;
    if (!visual || event.x === undefined || event.y === undefined) {
      return done();
    }
    const target = this.cellCenter(event.x, event.y);
    const squares = Math.max(
      Math.abs((event.from_x ?? event.x) - event.x),
      Math.abs((event.from_y ?? event.y) - event.y),
      1,
    );
    this.tweens.add({
      targets: visual.container,
      x: target.x,
      y: target.y,
      duration: squares * MOVE_MS_PER_SQUARE,
      ease: 'Sine.easeInOut',
      onComplete: () => done(),
    });
  }

  private animateFire(event: ApiBattleEvent, done: () => void) {
    const attacker = this.shipById(event.ship_id);
    const target = this.shipById(event.target_id);
    const attackerVisual = event.ship_id ? this.visuals.get(event.ship_id) : undefined;
    const targetVisual = event.target_id ? this.visuals.get(event.target_id) : undefined;
    if (!attacker || !target || !attackerVisual || !targetVisual) {
      return done();
    }
    const from = attackerVisual.container;
    const to = targetVisual.container;
    if (isMissileFire(event)) {
      return this.animateMissile(event, attacker, target, from, to, done);
    }
    const beam = this.add.graphics();
    beam.setDepth(20);
    const color = this.ownerColor(attacker.owner_id);
    beam.lineStyle(2.5, color, 0.95);
    beam.lineBetween(from.x, from.y, to.x, to.y);
    beam.lineStyle(5, color, 0.25);
    beam.lineBetween(from.x, from.y, to.x, to.y);
    this.fxLayer.add(beam);

    this.tweens.add({
      targets: beam,
      alpha: 0,
      duration: 280,
      delay: 90,
      onComplete: () => beam.destroy(),
    });
    this.showHitPopup(event, target, to, done);
  }

  /**
   * Missiles render as a small warhead with a fading exhaust trail that flies
   * from shooter to target; the damage popup (or MISS) follows on arrival.
   */
  private animateMissile(
    event: ApiBattleEvent,
    attacker: ApiTacticalShip,
    target: ApiTacticalShip,
    from: Phaser.GameObjects.Container,
    to: Phaser.GameObjects.Container,
    done: () => void,
  ) {
    const color = this.ownerColor(attacker.owner_id);
    const radius = Math.max(2.5, this.cellSize * 0.07);
    const warhead = this.add.graphics();
    warhead.setDepth(22);
    warhead.fillStyle(0xfff1c1, 1);
    warhead.fillCircle(0, 0, radius);
    warhead.lineStyle(1.5, color, 1);
    warhead.strokeCircle(0, 0, radius + 1);
    warhead.setPosition(from.x, from.y);
    this.fxLayer.add(warhead);

    const trail = this.add.graphics();
    trail.setDepth(21);
    this.fxLayer.add(trail);
    const puffs: { x: number; y: number; life: number }[] = [];

    const squares = Math.max(
      1,
      Math.max(Math.abs(attacker.x - target.x), Math.abs(attacker.y - target.y)),
    );
    this.tweens.add({
      targets: warhead,
      x: to.x,
      y: to.y,
      duration: Math.max(MISSILE_MIN_MS, squares * MISSILE_MS_PER_SQUARE),
      ease: 'Quad.easeIn',
      onUpdate: () => {
        puffs.push({ x: warhead.x, y: warhead.y, life: 1 });
        trail.clear();
        for (const puff of puffs) {
          puff.life -= 0.08;
          if (puff.life <= 0) continue;
          trail.fillStyle(0xffb26b, puff.life * 0.6);
          trail.fillCircle(puff.x, puff.y, radius * puff.life);
        }
      },
      onComplete: () => {
        warhead.destroy();
        this.tweens.add({
          targets: trail,
          alpha: 0,
          duration: 200,
          onComplete: () => trail.destroy(),
        });
        this.showHitPopup(event, target, to, done);
      },
    });
  }

  private showHitPopup(
    event: ApiBattleEvent,
    target: ApiTacticalShip,
    to: Phaser.GameObjects.Container,
    done: () => void,
  ) {
    const label =
      (event.damage ?? 0) > 0 ? `-${event.damage}` : 'MISS';
    const text = this.add
      .text(to.x, to.y - this.cellSize * 0.4, label, {
        fontFamily: '"Space Grotesk", monospace',
        fontSize: `${Math.max(13, Math.floor(this.cellSize * 0.32))}px`,
        color: (event.damage ?? 0) > 0 ? '#ffd166' : '#9fb8ff',
        stroke: '#000000',
        strokeThickness: 3,
      })
      .setOrigin(0.5)
      .setDepth(21);
    this.fxLayer.add(text);

    this.tweens.add({
      targets: text,
      y: text.y - this.cellSize * 0.5,
      alpha: 0,
      duration: 650,
      onComplete: () => {
        text.destroy();
        done();
      },
    });

    // reflect damage on the HP bar right away using authoritative state
    this.paintHpBar(target);
  }

  private animateDestroyed(event: ApiBattleEvent, done: () => void) {
    const visual = event.ship_id ? this.visuals.get(event.ship_id) : undefined;
    if (!visual) return done();
    const { x, y } = visual.container;

    const blast = this.add.graphics();
    blast.setDepth(22);
    this.fxLayer.add(blast);
    const steps = 5;
    let step = 0;
    const timer = this.time.addEvent({
      delay: 70,
      repeat: steps - 1,
      callback: () => {
        step++;
        blast.clear();
        const r = (this.cellSize * 0.55 * step) / steps;
        blast.fillStyle(0xff8a3d, 1 - step / (steps + 1));
        blast.fillCircle(x, y, r);
        blast.fillStyle(0xffe08a, 1 - step / (steps + 0.5));
        blast.fillCircle(x, y, r * 0.55);
        if (step >= steps) {
          blast.destroy();
          timer.remove();
        }
      },
    });

    this.tweens.add({
      targets: visual.container,
      alpha: 0,
      scale: 0.6,
      duration: 380,
      onComplete: () => {
        visual.container.destroy();
        this.visuals.delete(visual.shipId);
        done();
      },
    });
  }

  private animateFinished(event: ApiBattleEvent, done: () => void) {
    let label = 'BATTLE OVER';
    let color = 0xcfd8ff;
    if (event.draw) {
      label = 'STALEMATE — FLEETS DISENGAGE';
    } else if (event.winner_owner_id === this.config.playerId) {
      label = 'VICTORY';
      color = 0x7df0a8;
    } else {
      label = 'FLEET LOST';
      color = 0xff8a7d;
    }
    this.announce(label, color, done, 1400);
  }

  private announce(
    label: string,
    color: number,
    done: () => void,
    holdMs = 700,
  ) {
    const text = this.add
      .text(this.scale.width / 2, this.scale.height * 0.42, label, {
        fontFamily: '"Space Grotesk", monospace',
        fontSize: `${Math.max(22, Math.floor(this.scale.width / 28))}px`,
        color: `#${color.toString(16).padStart(6, '0')}`,
        stroke: '#000000',
        strokeThickness: 4,
        letterSpacing: 3,
      })
      .setOrigin(0.5)
      .setDepth(30)
      .setAlpha(0);
    this.tweens.add({
      targets: text,
      alpha: 1,
      duration: 180,
      hold: holdMs,
      yoyo: true,
      onComplete: () => {
        text.destroy();
        done();
      },
    });
  }

  /** Snap every visual to the authoritative server state after animations. */
  private syncToState() {
    for (const ship of this.battle.ships) {
      const visual = this.visuals.get(ship.id);
      if (!visual) continue;
      if (ship.destroyed) {
        visual.container.destroy();
        this.visuals.delete(ship.id);
        continue;
      }
      const { x, y } = this.cellCenter(ship.x, ship.y);
      visual.container.setPosition(x, y);
      this.paintHpBar(ship);
    }
  }
}
