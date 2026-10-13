# Sequence: App Discovery
**Requirements:** R141-144

**Source Spec:** app-discovery.md
**Use Case:** Startup ranks apps and registers viewdefs; activation changes re-run it

## Startup

```
Server              AppDiscovery              ViewdefManager             Config
  |                      |                          |                       |
1.1 |--Start: site dir exists? (no -> return error naming path)              |
  |                      |                          |                       |
1.2 |--Discover(site, builtins)-->|                   |                       |
  |   1.2.1 list lua/*, keep entries os.Stat says are dirs                   |
  |   1.2.2 rank: builtin tier, user tier, alphabetical within tier          |
  |<--[]App (Dir = site/lua/<app>)                  |                       |
  |                      |                          |                       |
1.3 |--ViewdefSources(apps)->|  app viewdefs..., then site viewdefs; missing dirs omitted
  |<--[]Source-----------|                          |                       |
  |                      |                          |                       |
1.4 |--LoadFromSources(sources)--------------------->|                       |
  |                      |   1.4.1 walk sources in rank order                 |
  |                      |   1.4.2 first source to offer a key registers it   |
  |                      |   1.4.3 later offer = collision: skip, Log(winner, loser)-->|
```

## Re-discovery (hot-loading)

```
FileSystem        ViewdefStore(hot)      LuaHotLoader         AppDiscovery        ViewdefManager
   |                    |                     |                     |                    |
2.1 |--create/remove lua/<app>-->|             |                     |                    |
   |                    |--Discover--------------------------------->|                    |
   |                    |--SetSources(new sources)-------------------------------------->|
   |                    |   2.1.1 re-run LoadFromSources; changed keys marked for re-push  |
   |                    |   2.1.2 watch each new source dir, drop watches of removed ones  |
2.2 |--same event------------------------>|                     |                    |
   |                    |                     |--Discover--------->|                    |
   |                    |                     |   2.2.1 add watches under new App.Dir    |
   |                    |                     |   2.2.2 drop watches of removed apps;    |
   |                    |                     |       loaded Lua stays loaded          |
```

## Listing collisions

```
Host (e.g. ark)           cli / Server              ViewdefManager               AppDiscovery
   |                           |                          |                            |
3.1 |--Server.ViewdefCollisions()->|--Collisions()-------->|                            |
   |                           |   3.1.1 copy `live`, sort by key then loser rank      |
   |<--[]ViewdefCollision------|<-------------------------|                            |
3.2 |--ResolveViewdefCollisions(site, builtins)----------->|                            |
   |                           |   3.2.1 Discover + ViewdefSources-------------------->|
   |                           |   3.2.2 fresh manager, silent log, LoadFromSources    |
   |                           |   3.2.3 return its Collisions() (same code as 1.4)    |
   |<--[]ViewdefCollision-------------------------------|                            |
```

## Notes

- Step 1.4.3 and every runtime losing offer record into the live collision set; the
  log dedupe is separate and lasts the whole run (R142).

- Paths stay under `lua/<app>`: watches, loads and events all use the entry
  path, so no target-to-link reverse mapping exists.
- A site `viewdefs/` appearing later is not watched for (R131); it is found at
  the next start.
