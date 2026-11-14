import { describe, expect, it } from 'vitest';

import { formatMissileDamage, selectedWeaponName, withWeapon } from '../ship-weapons';
import type { ApiTacticalBattle, ApiTacticalShip } from '../types/battle';
import {
  canShipFire,
  fireDisabledReason,
  formatMissileRack,
  isMissileFire,
  isOutOfMissiles,
  nearestEnemyShip,
} from '../types/battle';

const rack = (ammoLeft: number) => ({ name: 'Nuclear Missile', damage: 8, ammo_left: ammoLeft, ammo_max: 5 });

describe('missile HUD helpers', () => {
  it('formats the weapons row with damage and ammo left', () => {
    expect(formatMissileRack(rack(3))).toBe('Nuclear Missile · 8 dmg · 3/5 left');
    expect(formatMissileRack(rack(0))).toBe('Nuclear Missile · 8 dmg · 0/5 left');
  });

  it('disables fire with "Out of missiles" once the magazine is empty', () => {
    const loaded = { missiles: [rack(1)], has_fired: false };
    const empty = { missiles: [rack(0)], has_fired: false };
    expect(canShipFire(loaded)).toBe(true);
    expect(fireDisabledReason(loaded)).toBe('');
    expect(isOutOfMissiles(empty)).toBe(true);
    expect(canShipFire(empty)).toBe(false);
    expect(fireDisabledReason(empty)).toBe('Out of missiles');
  });

  it('keeps beam ships and old battles without missile fields firing as before', () => {
    const beamShip = { beams: [{ name: 'Laser', minDamage: 1, maxDamage: 4 }], has_fired: false };
    expect(isOutOfMissiles(beamShip)).toBe(false);
    expect(canShipFire(beamShip)).toBe(true);
    expect(canShipFire({ ...beamShip, has_fired: true })).toBe(false);
    expect(fireDisabledReason({ has_fired: false })).toBe('No weapons');
  });

  it('tells missile fire events apart from beam fire', () => {
    expect(isMissileFire({ type: 'fire', weapon_kind: 'missile' })).toBe(true);
    expect(isMissileFire({ type: 'fire' })).toBe(false);
    expect(isMissileFire({ type: 'move' })).toBe(false);
  });

  it('picks the nearest living enemy for the Fire button', () => {
    const ship = (id: number, owner: number, x: number, destroyed = false) =>
      ({ id, owner_id: owner, x, y: 0, destroyed }) as ApiTacticalShip;
    const shooter = ship(1, 1, 0);
    const battle = {
      ships: [shooter, ship(2, 2, 9), ship(3, 2, 4, true), ship(4, 2, 6), ship(5, 1, 1)],
    } as ApiTacticalBattle;
    expect(nearestEnemyShip(battle, shooter)?.id).toBe(4);
  });
});

describe('ship editor weapon slot', () => {
  const missiles = [{ name: 'Nuclear Missile' }];

  it('holds a beam or a missile, never both', () => {
    const armed = withWeapon({ beam: 'Laser', missile: '' }, 'Nuclear Missile', missiles);
    expect(armed).toEqual({ beam: '', missile: 'Nuclear Missile' });
    expect(selectedWeaponName(armed)).toBe('Nuclear Missile');
    expect(withWeapon(armed, 'Laser', missiles)).toEqual({ beam: 'Laser', missile: '' });
    expect(withWeapon(armed, '', missiles)).toEqual({ beam: '', missile: '' });
  });

  it('shows missile damage with shots per battle', () => {
    expect(formatMissileDamage({ damage: 8, ammo: 5 })).toBe('8 (5 shots/battle)');
  });
});
