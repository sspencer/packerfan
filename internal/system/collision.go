package system

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/event"
)

// CollisionSystem checks AABB overlaps between the player and ghosts and emits
// GhostEaten or PlayerEaten events.
type CollisionSystem struct {
	playerQuery *donburi.Query
	ghostQuery  *donburi.Query
}

func NewCollisionSystem() *CollisionSystem {
	return &CollisionSystem{
		playerQuery: donburi.NewQuery(
			filter.Contains(component.PlayerTag, component.Actor, component.Transform),
		),
		ghostQuery: donburi.NewQuery(
			filter.Contains(component.GhostTag, component.Actor, component.Transform, component.Ghost),
		),
	}
}

func (s *CollisionSystem) Update(w donburi.World) {
	playerEntry, ok := s.playerQuery.First(w)
	if !ok {
		return
	}

	playerActor := component.Actor.Get(playerEntry)
	playerTransform := component.Transform.Get(playerEntry)

	gameState := component.MustFindGameState(w)
	if gameState.FruitActive && SameTileCollide(playerTransform.Pos, gameState.FruitPos) {
		pts := gameState.FruitPoints
		if pts <= 0 {
			pts = int(gameState.Spec.BonusPoints)
		}
		event.FruitEaten.Publish(w, event.FruitEatenEvent{
			Points: pts,
		})
	}

	for ghostEntry := range s.ghostQuery.Iter(w) {
		ghostActor := component.Actor.Get(ghostEntry)
		ghostTransform := component.Transform.Get(ghostEntry)
		ghost := component.Ghost.Get(ghostEntry)

		if !SameTileCollide(playerTransform.Pos, ghostTransform.Pos) {
			continue
		}

		if ghost.State == config.Frightened && !ghostActor.IsEaten {
			event.GhostEaten.Publish(w, event.GhostEatenEvent{
				Entity: ghostEntry.Entity(),
				ID:     ghostActor.ID,
				X:      ghostTransform.Pos.X,
				Y:      ghostTransform.Pos.Y,
			})
			break
		}

		if (ghost.State == config.Chase || ghost.State == config.Scatter) && !playerActor.IsEaten {
			event.PlayerEaten.Publish(w, event.PlayerEatenEvent{
				Entity: playerEntry.Entity(),
				ID:     playerActor.ID,
				X:      playerTransform.Pos.X,
				Y:      playerTransform.Pos.Y,
			})
			break
		}
	}
}

func (s *CollisionSystem) updateCached(w donburi.World, c *worldCache) {
	if len(c.players) == 0 {
		return
	}
	p := &c.players[0]
	playerPos := p.transform.Pos
	playerTile := playerPos.ToCenterTile()

	gameState := c.gs
	if gameState.FruitActive && playerTile == gameState.FruitPos.ToCenterTile() {
		pts := gameState.FruitPoints
		if pts <= 0 {
			pts = int(gameState.Spec.BonusPoints)
		}
		ev := event.FruitEatenEvent{Points: pts}
		if c.bus != nil {
			event.FruitEaten.PublishTo(c.bus, ev)
		} else {
			event.FruitEaten.Publish(w, ev)
		}
	}

	for i := range c.ghosts {
		g := &c.ghosts[i]
		if playerTile != g.transform.Pos.ToCenterTile() {
			continue
		}

		if g.ghost.State == config.Frightened && !g.actor.IsEaten {
			ev := event.GhostEatenEvent{
				Entity: g.entity,
				ID:     g.actor.ID,
				X:      g.transform.Pos.X,
				Y:      g.transform.Pos.Y,
			}
			if c.bus != nil {
				event.GhostEaten.PublishTo(c.bus, ev)
			} else {
				event.GhostEaten.Publish(w, ev)
			}
			break
		}

		if (g.ghost.State == config.Chase || g.ghost.State == config.Scatter) && !p.actor.IsEaten {
			ev := event.PlayerEatenEvent{
				Entity: p.entity,
				ID:     p.actor.ID,
				X:      playerPos.X,
				Y:      playerPos.Y,
			}
			if c.bus != nil {
				event.PlayerEaten.PublishTo(c.bus, ev)
			} else {
				event.PlayerEaten.Publish(w, ev)
			}
			break
		}
	}
}

// SameTileCollide reports whether two actors occupy the same center-aligned
// 8x8 maze tile, matching the original arcade Pac-Man collision rule.
func SameTileCollide(pPos, gPos config.Vector) bool {
	return pPos.ToCenterTile() == gPos.ToCenterTile()
}

// Collide reports whether two 16x16 sprites overlap within their inner 6x6 core
// hitbox (5px inset per side). This triggers slightly before exact center so
// adjacent-tile grazes do not count as hits while preventing head-on pass-throughs.
func Collide(px, py, gx, gy float32) bool {
	const inset = float32(5)
	const box = float32(config.Sprite) - 2.0*inset

	pLeft := px + inset
	pTop := py + inset
	pRight := pLeft + box
	pBottom := pTop + box

	gLeft := gx + inset
	gTop := gy + inset
	gRight := gLeft + box
	gBottom := gTop + box

	return pLeft < gRight && pRight > gLeft && pTop < gBottom && pBottom > gTop
}
