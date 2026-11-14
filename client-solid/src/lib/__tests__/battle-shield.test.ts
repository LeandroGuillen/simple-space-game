import { describe, expect, it } from 'vitest';

import { formatShieldStrength } from '../types/battle';

describe('formatShieldStrength', () => {
  it('shows the shield class and per-hit reduction', () => {
    expect(formatShieldStrength({ shield_name: 'Class III', shield_reduce_damage: 3 })).toBe(
      'Class III (-3/hit)',
    );
    expect(formatShieldStrength({ shield_name: 'Class X', shield_reduce_damage: 10 })).toBe(
      'Class X (-10/hit)',
    );
  });

  it('shows None for unshielded ships', () => {
    expect(formatShieldStrength({ shield_name: 'None' })).toBe('None');
  });

  it('treats battles without shield fields as unshielded', () => {
    expect(formatShieldStrength({})).toBe('None');
  });
});
