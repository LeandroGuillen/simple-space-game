# TODO

## Race picks

- Spying and Stealthy Ships are disabled for new designs until espionage and detection have playable effects. Existing game saves retain their race data.
- Low-G and High-G planetary output and homeworld gravity work; their ground-combat effects remain unimplemented.
- Lucky currently provides a 35% chance of one research point per turn; random-event effects remain unimplemented.
- Ground Combat applies its damage multiplier during ground combat.

## Diplomacy

The diplomacy card is split into 4 slices: (1) first contact and the Diplomacy screen, (2) Declare War, (3) war/peace change combat and invasion, (4) Offer Peace and Server AI policy. Trade and non-aggression treaties are out of scope (war and peace only).

- Done: first contact, the Diplomacy screen (web client) and Declare War as a turn order. War is recorded on both sides, shown as "At war" and in the Turn Report, and ends when either empire is eliminated. Relation is always 0.
- Slice 3: battles and invasions still happen whenever rivals share a star, at war or not. Gating combat on war (and auto-declaring war when a fight would start in peacetime) is next.
- Slice 4: no way to make peace yet (Offer Peace, pending offers between humans, AI peace policy, relation modifiers). The Server AI ignores diplomacy and attacks everyone.
- The Odin client (`client-odin/`) does not show contacts or the Diplomacy screen yet, and still lists every player.
