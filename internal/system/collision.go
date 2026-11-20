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
	px := float32(playerTransform.Pos.X)
	py := float32(playerTransform.Pos.Y)

	for ghostEntry := range s.ghostQuery.Iter(w) {
		ghostActor := component.Actor.Get(ghostEntry)
		ghostTransform := component.Transform.Get(ghostEntry)
		ghost := component.Ghost.Get(ghostEntry)

		gx := float32(ghostTransform.Pos.X)
		gy := float32(ghostTransform.Pos.Y)

		if !Collide(px, py, gx, gy) {
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
