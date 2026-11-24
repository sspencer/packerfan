package audio

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/event"
)

// RegisterAudioEventHandlers subscribes gameplay events to sound effects on player.
func RegisterAudioEventHandlers(w donburi.World, player Player) {
	if player == nil {
		return
	}

	event.DotEaten.Subscribe(w, func(_ donburi.World, _ event.DotEatenEvent) {
		player.Play(SoundChomp)
	})
	event.PowerUpEaten.Subscribe(w, func(_ donburi.World, _ event.PowerUpEatenEvent) {
		player.Play(SoundChomp)
	})
	event.GhostEaten.Subscribe(w, func(_ donburi.World, _ event.GhostEatenEvent) {
		player.Play(SoundEatGhost)
	})
	event.FruitEaten.Subscribe(w, func(_ donburi.World, _ event.FruitEatenEvent) {
		player.Play(SoundEatFruit)
	})
	event.FruitBounced.Subscribe(w, func(_ donburi.World, _ event.FruitBouncedEvent) {
		player.Play(SoundFruitBounce)
	})
	event.ExtraLifeAwarded.Subscribe(w, func(_ donburi.World, _ event.ExtraLifeAwardedEvent) {
		player.Play(SoundExtraLife)
	})
}
