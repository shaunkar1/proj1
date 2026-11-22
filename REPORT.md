# Helper D2 — the opening

Harness: `sim -mode practice -n 40 -j 2 -ramp 0.44 -aggro 0`, candidate
`prof:f75a220+g+s+d100+x3` (the q7 reference profile). Local control reproduces the
stated baseline: **kills 5.45 / 5.58 / 4.70** on three seed blocks (stated 5.57).
"kills" in the sim output is kills before our own death, i.e. the practice score.

## 1. How the opening actually works

- `b.farmPhase` (bot/strat.go:363) = `PracFarmCubes > 0 && !sawOtherDeath && cubes < 10 && t < 75`.
  While set, practice answers only a mech that has hit us within 4 s and is inside 100 m
  (strat.go:1988-1991). So the opening is deliberately pacifist for 75 s or 10 cubes.
- The opening's *movement* is driven entirely by `pickObjective` (strat.go:2198), a
  value-minus-distance auction over cubes, crates, roof loot, drones, weapons and supply
  drops. `PracNearFarm` is on in this profile, which only makes the distance term steeper
  (`k = 1.3`, strat.go:2465). There is no term anywhere in that auction for how many
  hostiles are near **us** — `enemiesNear` is passed in but is used only to gate roof
  loot, drones and exploration. `PracSqD` ("sq<D>") gates loot that is near hostiles, which
  is about where the *cube* is, not about where *we* are.
- **`isOpening` (strat.go:3143) is not the game opening and does not run in practice.**
  It is a first-strike rule, gated on `!b.ffaSure || b.practice` in its first line. Same for
  `lateStrike`. Ignore both for this work.

## 2. Measurements (40 matches, seeds 9500000+, 2 Hz timeline via `-plog`)

Death buckets: `<80 s` 2% (2.00 kills), `80-120 s` 25% (3.20), `120-160 s` 20% (6.50),
`>=160 s / alive` 52% (6.29). Matches the stated profile; the loss is all before 120 s.

Early-death (died < 120 s, n=11) vs survivors (>= 160 s, n=21), mean over each window:

| window | hostiles < 200 m | hostiles < 120 m | dist from spawn | dist from centre | farm % of ticks |
|---|---|---|---|---|---|
| 0-20 s   | 2.25 / 2.17 | 0.78 / 0.74 | 78 / 66 m  | 325 / 311 m | 58 / 53 |
| 20-40 s  | 3.81 / 2.97 | 1.59 / 1.38 | 137 / 140 m | 298 / 286 m | 21 / 22 |
| 40-60 s  | **4.36 / 3.27** | 2.02 / 1.35 | 134 / 188 m | 261 / 270 m | 12 / 18 |
| 60-80 s  | **4.96 / 3.35** | 2.57 / 1.30 | 147 / 205 m | 272 / 271 m | 12 / 12 |
| 80-100 s | **5.00 / 2.35** | 2.54 / 1.02 | 191 / 234 m | 227 / 282 m | 63 / 25 |
| 100-120 s| **5.26 / 2.90** | 3.13 / 1.38 | 276 / 256 m | 207 / 266 m | 61 / 19 |

(The 80-120 s rows for the early-death group are thin — only the matches still alive in
that window contribute, a handful — so read 40-80 s as the solid part.)

Per-match, first 120 s:

| group | n | first hostile < 200 m | max hostiles < 200 m | cubes@120 | farm % | kills |
|---|---|---|---|---|---|---|
| all | 40 | t = 0.0 s (100% of matches) | 6.75 | 7.58 | 28.2 | 5.45 |
| early death (<120 s) | 11 | t = 0.0 s | **7.55** | 4.36 | 34.0 | 3.09 |
| survive >= 160 s | 21 | t = 0.0 s | **6.24** | 8.81 | 25.0 | 6.29 |

Correlations (per match, first-120 s feature vs outcome):

| feature | vs survive >= 160 s | vs kills | vs death time |
|---|---|---|---|
| max hostiles < 200 m | **-0.348** | +0.019 | **-0.314** |
| farm fraction | -0.265 | +0.068 | -0.186 |
| dist from spawn @120 s | +0.261 | +0.267 | +0.285 |
| cubes @120 s | +0.241 | **+0.614** | +0.260 |

Mode census, first 120 s: roam 22.9, fight 15.3, cube 8.9, shy 8.0, weapon 7.5, zone 6.0,
crate 5.7, drop 5.0, grab 4.6, npc 3.2, avoid 3.1, retreat 2.1, heal 2.1, npcgo 1.9 %.
So roughly 28% of the opening is loot errands and 21% is fighting.

### What the data says, and where the stated hypothesis was wrong

1. **The crowd does not converge — it is there from t = 0.** A hostile is inside 200 m at the
   first sample in 40/40 matches. There is no quiet window to protect.
2. **The size of the crowd near us is the best single predictor of early death** I measured
   (r = -0.35 against surviving to 160 s). Early-death matches carry ~1.5x the hostiles
   within 200 m through 40-80 s.
3. **The bot does not react to it at all.** Over 40-80 s the farm rate is the same (12-18%)
   whether we are in a 3-mech neighbourhood or a 5-mech one. That is the real defect — not
   the farm schedule itself, but that the schedule is blind to the crowd.
4. **"Reckless *near spawn*" is wrong, and backwards.** Survivors are *further* from spawn at
   40-80 s (188/205 m vs 134/147 m) and distance from spawn correlates *positively* with
   surviving (+0.26). What the losers do is drift toward the arena centre (325 -> 207 m from
   centre by 120 s, while survivors hold 266-311 m).
5. **Cubes are mostly an effect, not a cause.** cubes@120 correlates +0.61 with kills but only
   +0.24 with surviving, and farming correlates *negatively* with surviving. Kills drop cubes,
   so the causal arrow runs the other way from what the farm schedule assumes.

## 3. The change: `oc<N>` — a crowd-triggered opening

New Params fields (bot/bot.go), new file bot/opend2.go, token in bot/profile.go.
**Default off** (`PracOpenCrowdN = 0`); practice-only.

While `t < PracOpenCrowdT` (default 90 s) and at least `N` non-NPC, non-ghost tracks seen
within `PracOpenCrowdA` s (default 10) stand within `PracOpenCrowdD` m (default 200):

- **loot stops being an errand**: `consider` drops `cube`, `crate`, `roof`, `npc`, `npcgo`
  and `site` objectives. Weapons, the laser rush and supply drops are deliberately left
  alone — laser pickup at ~35 s is where the kills come from, and blocking it would be a
  different (and worse) experiment.
- **if nothing is left to walk to**, a new mode `spread` walks to a point inside the closing
  circle that is further from the *centroid* of the known crowd than we are, with a bonus for
  room from the nearest single one of them; held 4 s so we commit.

Tokens: `+oc<N>` turns it on, `+ocd<D>` `+oct<T>` `+oca<A>` override the radius, window and
sighting age. Trigger counts print from `cmd/sim`.

## 4. Control vs variant — same batch, same seeds

Same binary, two arms run back to back over identical seed sets, 40 matches each arm per block.

| seed block | control kills | `+oc3` kills | diff | condition-true decisions | `spread` goals chosen |
|---|---|---|---|---|---|
| 9500000 | 5.45 | 6.28 | +0.83 | 4180 | 64 |
| 9600000 | 5.58 | 6.03 | +0.45 | 6225 | 104 |
| 9700000 | 4.70 | 5.17 | +0.47 | 6418 | 96 |
| **pooled (n=120 each)** | **5.24** | **5.83** | **+0.58** | 16823 | 264 |

Per-block diffs +0.83 / +0.45 / +0.47: mean +0.583, sd 0.214, se 0.123, t = 4.7 on 2 df
(p ~ 0.04). Three independent blocks, all positive, each paired on seeds.

Secondary movements, pooled: life 160 -> 167 s, damage dealt 10848 -> 11726,
cubes held 9.1 -> 10.9, zone damage 66 -> 23.

### Trigger counts — read this part

The rule **fires constantly** (16823 condition-true decisions over 120 matches; my offline
estimate from the control timelines was ~37% of decisions in the first 90 s, consistent; note
the counter is incremented at both call sites, so it over-counts decisions where the move
switch also reached the rule — treat it as "the rule fires constantly", not as an exact rate).
But **`spread` was chosen as the goal only 264 times — about 2 per match, ~8 s of walking.**
So essentially all of the measured gain comes from the **farm skip**, not from the
centroid repositioning. The repositioning is close to inert because `pickObjective` almost
always still returns *something* (a weapon or a drop), so the `ob == nil` branch is rarely
reached. I did not have time to test the two halves separately; that is the obvious next run
(a `+oc3` with the `spread` branch disabled, or vice versa).

## 5. Verdict

**This worked, and by more than the target**: +0.58 kills pooled over 120 paired matches per
arm, against a +0.25 requirement and a stated sd of 0.117 at 96 matches. It is not a threshold
nudge — it adds a term the opening never had (how crowded *our own* neighbourhood is) and
suspends the whole loot auction on it.

Caveats I would not ship without resolving:
- **n is small.** 120 matches per arm, three blocks. The effect is consistent in sign and
  comfortably bigger than block-to-block drift in these runs, but the team's own convention is
  400-1200 matches. Re-run at 400+ before trusting +0.58 as the size of the effect; I would
  expect the true value to be smaller than the point estimate.
- **The named mechanism is not the mechanism.** The centroid `spread` move barely runs. If
  someone reports this as "position by distance from the hostile centroid works", that is wrong;
  what works is "stop farming while the neighbourhood is crowded". Worth separating before
  spending more parameter search on the `spread` goal.
- Untested interactions with `PracSqD` ("sq"), `PracDropCrowd` ("dc") and `PracCubeMax` ("m"),
  which all touch the same auction.
- `cubes@120` rose rather than fell, which is the opposite of the naive expectation and supports
  the "cubes come from kills" reading; but it also means the farm schedule's stated purpose
  (reach 10 cubes) is still being met, so there may be room to suspend farming harder.

Suggested next runs, in order: (a) 400 matches per arm at `+oc3` to pin the size;
(b) `+oc2` / `+oc4` and `+ocd160` / `+ocd240` to find the shape; (c) farm-skip alone vs
`spread` alone to attribute the effect.
