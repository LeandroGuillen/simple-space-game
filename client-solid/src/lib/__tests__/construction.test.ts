import { describe, expect, it } from 'vitest';

import { isAutobuildProject } from '../construction';

describe('isAutobuildProject', () => {
  it('excludes colonization, ship, android, and spy projects', () => {
    expect(isAutobuildProject({ name: 'Automated Factory' })).toBe(true);
    expect(isAutobuildProject({ name: 'Research Lab' })).toBe(true);

    expect(isAutobuildProject({ name: 'Colony Base' })).toBe(false);
    expect(isAutobuildProject({ name: 'Colony Ship' })).toBe(false);
    expect(isAutobuildProject({ name: 'Outpost Ship' })).toBe(false);
    expect(isAutobuildProject({ name: 'Transport' })).toBe(false);
    expect(isAutobuildProject({ name: 'Freighters' })).toBe(false);
    expect(isAutobuildProject({ name: 'Spy' })).toBe(false);
    expect(isAutobuildProject({ name: 'Android Farmers' })).toBe(false);
    expect(isAutobuildProject({ name: 'Android Workers' })).toBe(false);
    expect(isAutobuildProject({ name: 'Android Scientists' })).toBe(false);
  });

  it('excludes future ship projects by name', () => {
    expect(isAutobuildProject({ name: 'Scout Ship' })).toBe(false);
  });
});
