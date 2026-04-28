## Population Growth

Each owned colony's `Population` changes per turn by the value returned by
`PopulationGrowth(p)` in `population.go`. The formula is intentionally simple
and additive so future systems can layer on top:

    growth = round(basePopGrowth * gravityFactor * habitabilityFactor)
           + (future modifiers)

- `basePopGrowth = 40` is the baseline. It is **not** a maximum — buildings,
  technologies, race traits, etc. are expected to push the result above 40
  later on.
- `gravityFactor` depends on the planet's gravity:
  Low `0.75`, Normal `1.0`, High `0.5`.
- `habitabilityFactor` depends on the planet's environment, ranging from
  `0.0` for uninhabitable environments (gas giant, asteroid belt) up to
  `1.0` for Terran/Gaia, with intermediate values for Toxic, Radiated,
  Barren, Desert, Tundra, Arid, Swamp and Ocean.
- The result can be **negative** once future modifiers are wired in
  (starvation, plagues, etc.) — this represents people dying.
- `Population` itself is clamped to `[0, MaxPop(p)]` inside `planetTurn`,
  so a positive growth past the cap is truncated and a negative growth
  past zero stops at zero.
- Unowned planets and empty colonies (`Population == 0`) are not processed.

Future work:
- Apply `model.Race.Stats.PopulationGrowth` once races are wired up.
- Apply building/tech bonuses (cloning center, etc.).
- Subtract starvation losses once a food/farming system exists.

### Reference (MoO 2)

For historical reference, MoO 2's formula was:

> The 'base growth rate' is `((number of colonists) * (space left on planet) * 1000)^0.5`
> (basically ranges from ~40 to ~80). Then this number is modified by things
> like race growth rate, starvation, etc. Finally, +100 is added for a
> cloning center. Multi-species colonies have different growth rates for
> each species. Starvation causes -50 growth per missing food.

We intentionally diverge from this: our growth is bounded by gravity and
habitability rather than by current vs. max population, which keeps growth
predictable until we layer on the modifiers above.
