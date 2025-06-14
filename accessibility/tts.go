package accessibility

import (
	"fmt"
	// "github.com/go-tts/tts/pkg/audio"
	// "github.com/go-tts/tts/pkg/speech"
	// htgotts "github.com/hegedustibor/htgo-tts"
	// handlers "github.com/hegedustibor/htgo-tts/handlers"
	// voices "github.com/hegedustibor/htgo-tts/voices"
)

func SpeakText(text string) {
	fmt.Println("SPEAK:", text)

	// speech := htgotts.Speech{Folder: "audio", Language: voices.English, Handler: &handlers.Native{}}
	// err := speech.Speak(text)
	// if err != nil {
	// 	fmt.Println("Could not play tts:", err)
	// 	return
	// }

	// audioIn, err := speech.FromText(text, speech.LangEn)
	// if err != nil {
	// 	fmt.Println("Could not create tts:", err)
	// 	return
	// }
	// err = audio.NewSpeaker().Play(audioIn)
	// if err != nil {
	// 	fmt.Println("Could not play tts:", err)
	// 	return
	// }
}
