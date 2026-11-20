package system

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/audio"
	"pacfan/internal/event"
)

// RegisterAudioEventHandlers subscribes gameplay events to sound effects on player.
func RegisterAudioEventHandlers(w donburi.World, player audio.Player) {
	if player == nil {
		return
	}

	event.DotEaten.Subscribe(w, func(_ donburi.World, _ event.DotEatenEvent) {
		player.Play(audio.SoundChomp)
	})
	event.PowerUpEaten.Subscribe(w, func(_ donburi.World, _ event.PowerUpEatenEvent) {
		player.Play(audio.SoundChomp)
	})
	event.GhostEaten.Subscribe(w, func(_ donburi.World, _ event.GhostEatenEvent) {
		player.Play(audio.SoundEatGhost)
	})
	event.FruitEaten.Subscribe(w, func(_ donburi.World, _ event.FruitEatenEvent) {
		player.Play(audio.SoundEatFruit)
	})
	event.ExtraLifeAwarded.Subscribe(w, func(_ donburi.World, _ event.ExtraLifeAwardedEvent) {
		player.Play(audio.SoundExtraLife)
	})
}
