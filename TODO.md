# TODO

## Race picks

- Spying and Stealthy Ships are disabled for new designs until espionage and detection have playable effects. Existing game saves retain their race data.
- Low-G and High-G planetary output and homeworld gravity work; their ground-combat effects remain unimplemented.
- Lucky currently provides a 35% chance of one research point per turn; random-event effects remain unimplemented.
- Ground Combat applies its damage multiplier during ground combat.

## Diplomacy

The diplomacy card is split into 4 slices: (1) first contact and the Diplomacy screen, (2) Declare War, (3) war/peace change combat and invasion, (4) Offer Peace and Server AI policy. Trade and non-aggression treaties are out of scope (war and peace only).

- Done: first contact, the Diplomacy screen (web client) and Declare War as a turn order. War is recorded on both sides, shown as "At war" and in the Turn Report, and ends when either empire is eliminated. Relation is always 0.
- Done (slice 3): war and peace gate combat and invasion. A contact can carry a `peace_treaty` flag (both sides; omitted means no treaty). At war: fights and invasions as before. Default no-treaty state: a battle that would start at a shared star, or an Invade landing, auto-declares war first (`declared_war` events for both sides, `war_since` this turn), then proceeds. Under a peace treaty: no battle (fleets share orbit, treaty partners sit out multi-empire fights and do not contest orbit), Invade is rejected at submit and disabled in the web client with an explanation, and the Server AI plans no attacks or invasions against that empire. Declaring war (or an auto-declaration) clears the treaty.
- Invade choice: Invade in the default no-treaty state auto-declares war, then proceeds; Invade is rejected only while a peace treaty is in effect (the plan's "invade is rejected when not at war" is that post-treaty case).
- Slice 4: no way to make peace yet (Offer Peace, pending offers between humans, AI peace policy, relation modifiers). Accepting peace only has to set `peace_treaty` on both contacts (and clear `at_war`); the gate already honors it. The Diplomacy screen still labels the default no-treaty state "Peace" and does not show treaties. The Server AI attacks every empire it has no treaty with.
- The Odin client (`client-odin/`) does not show contacts or the Diplomacy screen yet, and still lists every player.
