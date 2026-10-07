package main

import (
	"log"
	"os"

	"gioui.org/app"

	"playerdoit/internal/ui/native"
)

func main() {
	go func() {
		if err := native.Run(os.Args[1:]); err != nil {
			log.Println("Aura Music Player:", err)
		}
		os.Exit(0)
	}()
	app.Main()
}
